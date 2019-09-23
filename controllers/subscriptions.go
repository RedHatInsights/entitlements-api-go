package controllers

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/RedHatInsights/platform-go-middlewares/identity"

	"github.com/karlseguin/ccache"
	"go.uber.org/zap"
)

type getter func(string) []string

var cache = ccache.New(ccache.Configure().MaxSize(500).ItemsToPrune(50))

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

var getSubscriptions = func(orgID string) types.SubscriptionsResponse {
	item := cache.Get(orgID)

	if item != nil && !item.Expired() {
		return types.SubscriptionsResponse{
			StatusCode: 200,
			Data:       item.Value().([]string),
			CacheHit:   true,
		}
	}

	smartManagementChecks := "SVC3124,RH00068,"
	ansibleChecks := "MCT3691,MCT3692,MCT3693,MCT3694,MCT3695,MCT3696"

	resp, err := getClient().Get(config.GetConfig().Options.GetString(config.Keys.SubsHost) +
		"/svcrest/subscription/v5/searchnested/criteria" +
		";web_customer_id=" + orgID +
		";sku=" + smartManagementChecks + ansibleChecks +
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
		skuStatus := skuInfo[1].Value
		// sku status == "" means it's a parent SKU
		if skuStatus == "" || skuStatus == "Active" || skuStatus == "Temporary" {
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

// Index the handler for GETs to /api/entitlements/v1/services/
func Index(getCall func(string) types.SubscriptionsResponse) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if getCall == nil {
			getCall = getSubscriptions
		}

		start := time.Now()
		reqCtx := req.Context().Value(identity.Key).(identity.XRHID).Identity
		res := getCall(reqCtx.Internal.OrgID)
		accNum := reqCtx.AccountNumber

		validAccNum := !(accNum == "" || accNum == "-1")

		if res.Error != nil {
			l.Log.Error("Unexpected error while talking to Subs Service", zap.Error(res.Error))
			http.Error(w, http.StatusText(500), 500)
			return
		}

		l.Log.Info("subs call complete",
			zap.Duration("subs_call_duration", time.Since(start)),
			zap.Bool("cache_hit", res.CacheHit),
		)

		if res.StatusCode != 200 {
			l.Log.Error("Got back a non 200 status code from Subscriptions Service",
				zap.Int("code", res.StatusCode),
				zap.String("body", res.Body),
			)

			http.Error(w, http.StatusText(500), 500)
			return
		}

		entitleInsights := validAccNum

		smartManagementSKUs := []string{"SVC3124", "RH00068"}
		entitleSmartManagement := len(checkCommonSkus(smartManagementSKUs, res.Data)) > 0

		ansibleSKUs := []string{"MCT3691", "MCT3692", "MCT3693", "MCT3694", "MCT3695", "MCT3696"}
		entitleAnsible := validAccNum && len(checkCommonSkus(ansibleSKUs, res.Data)) > 0

		entitleMigrations := validAccNum

		obj, err := json.Marshal(types.EntitlementsResponse{
			HybridCloud:     types.EntitlementsSection{IsEntitled: true}, //set to true until ready for hybrid entitlment checks to be enforced
			Insights:        types.EntitlementsSection{IsEntitled: entitleInsights},
			Openshift:       types.EntitlementsSection{IsEntitled: true},
			SmartManagement: types.EntitlementsSection{IsEntitled: entitleSmartManagement},
			Ansible:         types.EntitlementsSection{IsEntitled: entitleAnsible},
			Migrations:      types.EntitlementsSection{IsEntitled: entitleMigrations},
		})

		if err != nil {
			l.Log.Error("Unexpected error while unmarshalling JSON data from Subs Service", zap.Error(err))
			http.Error(w, http.StatusText(500), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(obj))
	}
}
