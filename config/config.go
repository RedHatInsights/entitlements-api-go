package config

import (
	"crypto/tls"
)

var config *EntitlementsConfig

type EntitlementsConfig struct {
	Certs *tls.Certificate
	Port string
}

func getPath(str string) string {
	return "/home/iphands/prog/insights/entitlements-meta/prod/" + str
	// return "/home/iphands/prog/cloud/enc/entitlements-meta/prod/" + str
}

func getCerts() *tls.Certificate {
	// Read the key pair to create certificate
	cert, err := tls.LoadX509KeyPair(getPath("prod-cert.crt"), getPath("prod-cert.key"))
	if err != nil { panic(err.Error()) }
	return &cert
}

func GetConfig() *EntitlementsConfig {
	if (config == nil) {
		config = &EntitlementsConfig {
			Certs: getCerts(),
			Port: ":3000",
		}
	}

	return config
}
