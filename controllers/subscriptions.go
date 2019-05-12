package controllers

import (
	"net/http"
	"github.com/go-chi/chi"
	"encoding/json"
	"cloud.redhat.com/entitlements/types"
)

func Subscriptions(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		obj, _ := json.Marshal(types.EntitlementsResponse{
			Hybrid_cloud:    types.EntitlementsSection{ Is_entitled: true },
			Insights:        types.EntitlementsSection{ Is_entitled: true },
			Openshift:       types.EntitlementsSection{ Is_entitled: true },
			Smart_mangement: types.EntitlementsSection{ Is_entitled: true },
		})
		w.Write([]byte(obj))
	})
}
