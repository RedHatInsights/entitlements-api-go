package middleware

import (
	"net/http"
	"context"
	"encoding/base64"
	"encoding/json"
	"cloud.redhat.com/entitlements/types"
)



func Identity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawHeaders := r.Header["X-Rh-Identity"]

		// must have an x-rh-id header
		if (len(rawHeaders) != 1) {
			panic("Fatal must include x-rh-id")
		}

		// must be able to base64 decode header
		idRaw, err := base64.StdEncoding.DecodeString(rawHeaders[0])
		if (err != nil) {
			panic(err)
		}

		var jsonData types.XRhIdentity
		err = json.Unmarshal(idRaw, &jsonData)
		if (err != nil) {
			panic(err)
		}

		if (jsonData.Account_number == "" || jsonData.Account_number == "-1") {
			panic("Invalid or missing account number")
		}

		if (jsonData.Internal.Org_id == "") {
			panic("Invalid or missing org_id")
		}

		ctx := context.WithValue(r.Context(), "org_id", jsonData.Internal.Org_id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
