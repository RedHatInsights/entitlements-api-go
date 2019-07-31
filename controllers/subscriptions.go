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

	resp, err := getClient().Get(config.GetConfig().Options.GetString(config.Keys.SubsHost) +
		"/svcrest/subscription/v5/search/criteria" +
		";web_customer_id=" + orgID +
		";sku=SVC3851,SVC3852,SVCSER0566,SVCSER0567,SVC3124" +
		";status=active;/options;products=ALL;")

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
	var subscriptionBody []types.SubscriptionBody
	json.Unmarshal(body, &subscriptionBody)

	for s := range subscriptionBody {
		skuValue := subscriptionBody[s].SubscriptionProducts
		for e := range skuValue {
			arr = append(arr, skuValue[e].Sku)
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
func checkCommon(skus []string, userSkus []string) []string {
	hash := make(map[string]bool)
	var common []string

	for sku := range skus {
		hash[skus[sku]] = true
	}

	for usku := range userSkus {
		if _, found := hash[userSkus[usku]]; found {
			common = append(common, userSkus[usku])
		}
	}

	return common
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

		entitleInsights := false

		if !(accNum == "" || accNum == "-1") {
			entitleInsights = true
		}

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

		hybridSKUs := []string{"SVC3851", "SVC3852", "SVCSER0566", "SVCSER0567"}
		entitleHybrid := len(checkCommon(hybridSKUs, res.Data)) > 0

		obj, err := json.Marshal(types.EntitlementsResponse{
			HybridCloud:    types.EntitlementsSection{IsEntitled: entitleHybrid},
			Insights:       types.EntitlementsSection{IsEntitled: entitleInsights},
			Openshift:      types.EntitlementsSection{IsEntitled: true},
			SmartMangement: types.EntitlementsSection{IsEntitled: (len(res.Data) > 0)},
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
