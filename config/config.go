package config

import (
	"fmt"
	"crypto/tls"
	"github.com/spf13/viper"
)

var config *EntitlementsConfig

type EntitlementsConfig struct {
	Certs *tls.Certificate
	Port string
	Options *viper.Viper
}

func getCerts(options *viper.Viper) *tls.Certificate {
	// Read the key pair to create certificate
	cert, err := tls.LoadX509KeyPair(
		options.GetString("CERT"),
		options.GetString("KEY"),
	)
	if err != nil { panic(err.Error()) }
	return &cert
}


func initialize() {
	var options *viper.Viper = viper.New()
	options.SetDefault("Port", "3000")
	options.SetEnvPrefix("ENT")
	options.AutomaticEnv()

	config = &EntitlementsConfig {
		Certs: getCerts(options),
		Options: options,
	}

	fmt.Println("- Starting Entitlements API -")
	fmt.Println("Will listen on " + options.GetString("PORT"))
}

func GetConfig() *EntitlementsConfig {
	if (config == nil) {
		initialize()
	}

	return config
}
