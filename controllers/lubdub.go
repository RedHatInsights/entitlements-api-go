package controllers

import (
	"net/http"
	"github.com/go-chi/chi"
)

func LubDub(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("lubdub"))
	})
}
