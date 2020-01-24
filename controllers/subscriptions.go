package controllers

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/RedHatInsights/platform-go-middlewares/identity"

	"github.com/karlseguin/ccache"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type getter func(string) []string

var cache = ccache.New(ccache.Configure().MaxSize(500).ItemsToPrune(50))

var bundleInfo []types.Bundle

func getClient() *http.Client {
	// Create a HTTPS client that uses the supplied pub/priv mutual TLS certs
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      config.GetConfig().RootCAs,
				Certificates: []tls.Certificate{*config.GetConfig().Certs},
			},
		},
	}
}

// SetBundleInfo sets the bundle information fetched from the YAML
func SetBundleInfo(yamlFilePath string) error {
	bundlesYaml, err := ioutil.ReadFile(yamlFilePath)

	if err != nil {
		return err
	}

	err = yaml.Unmarshal([]byte(bundlesYaml), &bundleInfo)
	if err != nil {
		return err
	}

	return nil
}

// GetSubscriptions calls the Subs service and returns the SKUs the user has
var GetSubscriptions = func(orgID string, skus string) types.SubscriptionsResponse {
	item := cache.Get(orgID)

	if item != nil && !item.Expired() {
		return types.SubscriptionsResponse{
			StatusCode: 200,
			Data:       item.Value().([]string),
			CacheHit:   true,
		}
	}

	resp, err := getClient().Get(config.GetConfig().Options.GetString(config.Keys.SubsHost) +
		"/svcrest/subscription/v5/searchnested/criteria" +
		";web_customer_id=" + orgID +
		";sku=" + skus +
		";/options;products=ALL/product.sku|product.statusCode")

	if err != nil {
		return types.SubscriptionsResponse{
			Error: err,
		}
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		return types.SubscriptionsResponse{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			Error:      nil,
			Data:       nil,
			CacheHit:   false,
		}
	}

	defer resp.Body.Close()
	var arr []string

	// Unmarshaling response from Subscription
	// Extracting skus and appending them to arr
	body, _ := ioutil.ReadAll(resp.Body)
	var SubscriptionDetails []types.SubscriptionDetails
	json.Unmarshal(body, &SubscriptionDetails)

	for s := range SubscriptionDetails {
		skuInfo := SubscriptionDetails[s].Entries
		skuName := skuInfo[0].Value
		skuStatus := strings.ToLower(skuInfo[1].Value)
		// sku status == "" means it's a parent SKU
		if skuStatus == "" || skuStatus == "active" || skuStatus == "temporary" {
			arr = append(arr, skuName)
		}
	}

	cache.Set(orgID, arr, time.Minute*10)

	return types.SubscriptionsResponse{
		StatusCode: resp.StatusCode,
		Data:       arr,
		CacheHit:   false,
	}
}

// Checks the common strings between two slices of strings and returns a slice of strings
// with the common skus
func checkCommonSkus(skus []string, userSkus []string) []string {
	skuHash := make(map[string]bool)
	var commonSKUs []string

	for sku := range skus {
		skuHash[skus[sku]] = true
	}

	for usku := range userSkus {
		if _, found := skuHash[userSkus[usku]]; found {
			commonSKUs = append(commonSKUs, userSkus[usku])
		}
	}

	return commonSKUs
}

func failOnDependencyError(errMsg string, res types.SubscriptionsResponse, w http.ResponseWriter) {
	dependencyError := types.DependencyErrorDetails{
		DependencyFailure: true,
		Service:           "Subscriptions Service",
		Status:            res.StatusCode,
		Endpoint:          config.GetConfig().Options.GetString(config.Keys.SubsHost),
		Message:           errMsg,
	}

	errorResponse := types.DependencyErrorResponse{Error: dependencyError}
	errorResponsejson, _ := json.Marshal(errorResponse)

	http.Error(w, string(errorResponsejson), 500)
}

// Index the handler for GETs to /api/entitlements/v1/services/
func Index() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var skus []string

		for b := range bundleInfo {
			skus = append(skus, bundleInfo[b].Skus...)
		}

		start := time.Now()
		reqCtx := req.Context().Value(identity.Key).(identity.XRHID).Identity
		res := GetSubscriptions(reqCtx.Internal.OrgID, strings.Join(skus, ","))
		accNum := reqCtx.AccountNumber

		validAccNum := !(accNum == "" || accNum == "-1")

		if res.Error != nil {
			errMsg := "Unexpected error while talking to Subs Service"
			l.Log.WithFields(logrus.Fields{"error": res.Error}).Error(errMsg)
			failOnDependencyError(errMsg, res, w)
			return
		}

		l.Log.WithFields(logrus.Fields{"subs_call_duration": time.Since(start), "cache_hit": res.CacheHit}).Info("subs call complete")

		if res.StatusCode != 200 {
			errMsg := "Got back a non 200 status code from Subscriptions Service"
			l.Log.WithFields(logrus.Fields{"code": res.StatusCode, "body": res.Body}).Error(errMsg)
			failOnDependencyError(errMsg, res, w)
			return
		}

		entitlementsResponse := make(map[string]types.EntitlementsSection)
		for b := range bundleInfo {
			entitle := true

			if len(bundleInfo[b].Skus) > 0 {
				entitle = validAccNum && len(checkCommonSkus(bundleInfo[b].Skus, res.Data)) > 0
			}

			if bundleInfo[b].UseValidAccNum {
				entitle = validAccNum && entitle
			}
			entitlementsResponse[bundleInfo[b].Name] = types.EntitlementsSection{IsEntitled: entitle}
		}

		obj, err := json.Marshal(entitlementsResponse)

		if err != nil {
			l.Log.WithFields(logrus.Fields{"error": err}).Error("Unexpected error while unmarshalling JSON data from Subs Service")
			http.Error(w, http.StatusText(500), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(obj))
	}
}
