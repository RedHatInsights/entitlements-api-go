package controllers

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
)

func getClient() *http.Client {
	cfg := config.GetConfig()
	options := cfg.Options
	timeout := options.GetInt(config.Keys.ITServicesTimeoutSeconds)

	// Create a HTTPS client that uses the supplied pub/priv mutual TLS certs
	return &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      cfg.RootCAs,
				Certificates: []tls.Certificate{*config.GetConfig().Certs},
			},
		},
	}
}
