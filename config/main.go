package config

import (
	"crypto/tls"
	"github.com/spf13/viper"
)

var config *EntitlementsConfig

// EntitlementsConfig is a global configuration struct for the API
type EntitlementsConfig struct {
	Certs   *tls.Certificate
	Port    string
	Options *viper.Viper
}

func getCerts(options *viper.Viper) *tls.Certificate {
	// Read the key pair to create certificate
	cert, err := tls.LoadX509KeyPair(
		options.GetString("CERT"),
		options.GetString("KEY"),
	)
	if err != nil {
		panic(err.Error())
	}
	return &cert
}

func initialize() {
	var options = viper.New()
	options.SetDefault("Port", "3000")
	options.SetDefault("SubsHost", "https://subscription.api.redhat.com")
	options.SetEnvPrefix("ENT")
	options.AutomaticEnv()

	config = &EntitlementsConfig{
		Certs:   getCerts(options),
		Options: options,
	}
}

// GetConfig provides a singleton global EntitlementsConfig instance
func GetConfig() *EntitlementsConfig {
	if config == nil {
		initialize()
	}

	return config
}
