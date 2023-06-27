package controllers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// LubDub responds back with a simple heartbeat
func LubDub(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("lubdub"))
	})
}
