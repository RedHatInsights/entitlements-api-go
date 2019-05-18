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

	resp, err := getClient().Get(config.GetConfig().Options.GetString("SubsHost") +
		"/svcrest/subscription/v5/search/criteria" +
		";web_customer_id=" + orgID +
		";sku=SVC3124" +
		";status=active")

	if err != nil {
		panic(err.Error())
	}
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

		l.Log.Error("Got back a non 200 status code from Subscriptions Service",
			zap.Int("code", resp.StatusCode),
			zap.String("body", string(body)),
		)

		return types.SubscriptionsResponse{
			StatusCode: resp.StatusCode,
			Data:       nil,
			CacheHit:   false,
		}
	}

	defer resp.Body.Close()
	var arr []string
	json.NewDecoder(resp.Body).Decode(&arr)
	cache.Set(orgID, arr, time.Minute*10)
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
		var res = getCall(req.Context().Value("org_id").(string))
		l.Log.Info("subs call complete",
			zap.Duration("subs_call_duration", time.Since(start)),
			zap.Bool("cache_hit", res.CacheHit),
		)

		if res.StatusCode != 200 {
			http.Error(w, http.StatusText(500), 500)
			return
		}

		obj, err := json.Marshal(types.EntitlementsResponse{
			HybridCloud:    types.EntitlementsSection{IsEntitled: true},
			Insights:       types.EntitlementsSection{IsEntitled: true},
			Openshift:      types.EntitlementsSection{IsEntitled: true},
			SmartMangement: types.EntitlementsSection{IsEntitled: (len(res.Data) > 0)},
		})

		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(obj))
	}
}
