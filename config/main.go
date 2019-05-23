package config

import (
	"crypto/tls"
	"github.com/spf13/viper"
	"crypto/x509"
	"io/ioutil"
	"fmt"
 )

var config *EntitlementsConfig

// EntitlementsConfig is a global configuration struct for the API
type EntitlementsConfig struct {
	Certs   *tls.Certificate
	RootCAs *x509.CertPool
	Port    string
	Options *viper.Viper
}

func getRootCAs() *x509.CertPool {
	const localCertFile = "./resources/ca.crt"

	// force the CA cert
	rootCAs, err := x509.SystemCertPool()
	if rootCAs == nil {
		panic("Could not load system CA certs")
	}

	if err != nil {
		panic(fmt.Sprintf("Could not load system CA certs: %v", err))
	}

	certs, err := ioutil.ReadFile(localCertFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to append %q to RootCAs: %v", localCertFile, err))
	}

	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		panic(fmt.Sprintf("Failed to AppendCertsFromPEM %q to RootCAs", localCertFile))
	}

	return rootCAs
}

func loadCerts(options *viper.Viper) (tls.Certificate, error){
	if (options.GetBool("CertsFromEnv")) {
		return tls.X509KeyPair(
			[]byte(options.GetString("CERT")),
			[]byte(options.GetString("KEY")),
		)
	}

	return tls.LoadX509KeyPair(options.GetString("CERT"), options.GetString("KEY"))
}

func getCerts(options *viper.Viper) *tls.Certificate {
	// Read the key pair to create certificate
	cert, err := loadCerts(options)

	if err != nil {
		panic(err.Error())
	}

	return &cert
}

func initialize() {
	var options = viper.New()
	options.SetDefault("CertsFromEnv", false)
	options.SetDefault("Port", "3000")
	options.SetDefault("SubsHost", "https://subscription.api.redhat.com")
	options.SetEnvPrefix("ENT")
	options.AutomaticEnv()

	config = &EntitlementsConfig{
		Certs:   getCerts(options),
		RootCAs: getRootCAs(),
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
