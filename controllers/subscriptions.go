package controllers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
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

// var checkHybrid = func(orgID string) bool {
// 	resp, err := getClient().Get(config.GetConfig().Options.GetString(config.Keys.SubsHost) +
// 		"/svcrest/subscription/v5/search/criteria" +
// 		";web_customer_id=" + orgID +
// 		";sku=SVC3851,SVC3852,SVCSER0566,SVCSER0567," +
// 		";status=active")

// 	if !(err == nil || resp.StatusCode == 200) {
// 		return false
// 	}

// 	var arr []string
// 	json.NewDecoder(resp.Body).Decode(&arr)

// 	if len(arr) > 0 {
// 		return true
// 	}

// 	return false
// }

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
		";status=active;/options;proucts=ONLY_MATCHING;/product.sku")

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

	body, _ := ioutil.ReadAll(resp.Body)
	var subscriptionBody []types.SubscriptionBody
	json.Unmarshal(body, &subscriptionBody)
	for s := range subscriptionBody {
		skuValue := subscriptionBody[s].Entries
		for e := range skuValue {
			//fmt.Printf("%v", skuValue[e].Value)
			arr = append(arr, skuValue[e].Value)
		}
		// fmt.Println()
	}

	cache.Set(orgID, arr, time.Minute*10)
	fmt.Println("array", len(arr))
	fmt.Println(arr)

	return types.SubscriptionsResponse{
		StatusCode: resp.StatusCode,
		Data:       arr,
		CacheHit:   false,
	}
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

		obj, err := json.Marshal(types.EntitlementsResponse{
			HybridCloud:    types.EntitlementsSection{IsEntitled: true},
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
