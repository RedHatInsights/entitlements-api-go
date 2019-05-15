package controllers

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/types"

	"github.com/go-chi/chi"
	"github.com/karlseguin/ccache"
)

var cache = ccache.New(ccache.Configure().MaxSize(500).ItemsToPrune(50))

func getClient() *http.Client {
	// Create a HTTPS client and supply the created CA pool and certificate
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{*config.GetConfig().Certs},
			},
		},
	}
}

func getSubscriptions(orgID string) []string {
	item := cache.Get(orgID)

	if item != nil && !item.Expired() {
		return item.Value().([]string)
	}

	resp, err := getClient().Get(config.GetConfig().Options.GetString("SubsHost") +
		"/svcrest/subscription/v5/search/criteria" +
		";web_customer_id=" + orgID +
		";sku=SVC3124" +
		";status=active")

	if err != nil {
		panic(err.Error())
	}
	defer resp.Body.Close()

	var arr []string
	json.NewDecoder(resp.Body).Decode(&arr)
	cache.Set(orgID, arr, time.Minute*10)
	return arr
}

func Subscriptions(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		var arr = getSubscriptions(req.Context().Value("org_id").(string))

		obj, err := json.Marshal(types.EntitlementsResponse{
			HybridCloud:    types.EntitlementsSection{IsEntitled: true},
			Insights:       types.EntitlementsSection{IsEntitled: true},
			Openshift:      types.EntitlementsSection{IsEntitled: true},
			SmartMangement: types.EntitlementsSection{IsEntitled: (len(arr) > 0)},
		})

		if err != nil {
			panic(err)
		}

		w.Write([]byte(obj))
	})
}
