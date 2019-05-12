package server

import (
	"net/http"
)

func Launch() {
	r := DoRoutes()
	http.ListenAndServe(":3000", r)
}
