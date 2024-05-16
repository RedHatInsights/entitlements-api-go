package controllers

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
)

var client *http.Client

func getClient() *http.Client {
	if client != nil {
		return client
	}

	cfg := config.GetConfig()
	timeout := cfg.Options.GetInt(config.Keys.ITServicesTimeoutSeconds)

	// Create a HTTPS client that uses the supplied pub/priv mutual TLS certs
	client = &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      cfg.RootCAs,
				Certificates: []tls.Certificate{*config.GetConfig().Certs},
			},
		},
	}

	return client
}
