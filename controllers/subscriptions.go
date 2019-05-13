package controllers

import (
	"crypto/tls"
	"net/http"
	"github.com/go-chi/chi"
	"encoding/json"
	"cloud.redhat.com/entitlements/types"
)

var certs *tls.Certificate

func getPath(str string) string {
	return "/home/iphands/prog/cloud/enc/entitlements-meta/prod/" + str
}

func getCerts() *tls.Certificate {
	if (certs == nil) {
		// Read the key pair to create certificate
		cert, err := tls.LoadX509KeyPair(getPath("prod-cert.crt"), getPath("prod-cert.key"))
		if err != nil { panic(err.Error()) }
		certs = &cert
	}

	return certs
}

func getClient() *http.Client {
	// Create a HTTPS client and supply the created CA pool and certificate
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{*getCerts()},
			},
		},
	}
}

func getSubscriptions() []string {
	// resp, err := http.Get("http://localhost:8000")

	resp, err := getClient().Get("https://subscription.api.redhat.com/svcrest/subscription/v5/search/criteria;web_customer_id=6340056;sku=SVC3124;status=active")
	if err != nil { panic(err.Error()) }
	defer resp.Body.Close()

	var arr []string
	json.NewDecoder(resp.Body).Decode(&arr)
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
