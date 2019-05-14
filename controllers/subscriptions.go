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

func getSubscriptions(orgId string) []string {
	item := cache.Get(orgId)

	if (item != nil && !item.Expired()) {
		return item.Value().([]string)
	}

	resp, err := getClient().Get(config.GetConfig().Options.GetString("SubsHost") +
		"/svcrest/subscription/v5/search/criteria" +
		";web_customer_id=" + orgId +
		";sku=SVC3124" +
		";status=active")

	if err != nil { panic(err.Error()) }
	defer resp.Body.Close()

	var arr []string
	json.NewDecoder(resp.Body).Decode(&arr)
	cache.Set(orgId, arr, time.Minute * 10)
	return arr
}

func Subscriptions(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		var arr []string = getSubscriptions(req.Context().Value("org_id").(string))

		obj, err := json.Marshal(types.EntitlementsResponse{
			Hybrid_cloud:    types.EntitlementsSection{ Is_entitled: true },
			Insights:        types.EntitlementsSection{ Is_entitled: true },
			Openshift:       types.EntitlementsSection{ Is_entitled: true },
			Smart_mangement: types.EntitlementsSection{ Is_entitled: (len(arr) > 0) },
		})

		if (err != nil) {
			panic(err)
		}

		w.Write([]byte(obj))
	})
}
