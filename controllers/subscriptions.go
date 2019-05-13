package controllers

import (
	"time"
	"crypto/tls"
	"net/http"
	"github.com/go-chi/chi"
	"encoding/json"
	"github.com/karlseguin/ccache"
	"cloud.redhat.com/entitlements/types"
	"cloud.redhat.com/entitlements/config"
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

const customerId string = "6340056"

func getSubscriptions() []string {
	item := cache.Get(customerId)

	if (item != nil && !item.Expired()) {
		return item.Value().([]string)
	}

	resp, err := getClient().Get("https://subscription.api.redhat.com" +
		"/svcrest/subscription/v5/search/criteria" +
		";web_customer_id=" + customerId +
		";sku=SVC3124" +
		";status=active")

	if err != nil { panic(err.Error()) }
	defer resp.Body.Close()

	var arr []string
	json.NewDecoder(resp.Body).Decode(&arr)
	cache.Set(customerId, arr, time.Minute * 10)
	return arr
}

func Subscriptions(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		var arr []string = getSubscriptions()

		obj, _ := json.Marshal(types.EntitlementsResponse{
			Hybrid_cloud:    types.EntitlementsSection{ Is_entitled: true },
			Insights:        types.EntitlementsSection{ Is_entitled: true },
			Openshift:       types.EntitlementsSection{ Is_entitled: true },
			Smart_mangement: types.EntitlementsSection{ Is_entitled: (len(arr) > 0) },
		})
		w.Write([]byte(obj))
	})
}
