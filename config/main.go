package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/spf13/viper"
)

var config *EntitlementsConfig

// EntitlementsConfig is a global configuration struct for the API
type EntitlementsConfig struct {
	Certs   *tls.Certificate
	RootCAs *x509.CertPool
	Port    string
	Options *viper.Viper
}

// EntitlementsConfigKeysType is the definition of the struct hat houses all the env variables key names
type EntitlementsConfigKeysType struct {
	Key             string
	Cert            string
	Port            string
	CertsFromEnv    string
	SubsHost        string
	CaPath          string
	OpenAPISpecPath string
	BundleInfoYaml  string
}

// Keys is a struct that houses all the env variables key names
var Keys = EntitlementsConfigKeysType{
	Key:             "KEY",
	Cert:            "CERT",
	Port:            "PORT",
	CertsFromEnv:    "CERTS_FROM_ENV",
	SubsHost:        "SUBS_HOST",
	CaPath:          "CA_PATH",
	OpenAPISpecPath: "OPENAPI_SPEC_PATH",
	BundleInfoYaml:  "BUNDLE_INFO_YAML",
}

func getRootCAs(localCertFile string) *x509.CertPool {
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

func loadCerts(options *viper.Viper) (tls.Certificate, error) {
	if options.GetBool("CERTS_FROM_ENV") == true {
		return tls.X509KeyPair(
			[]byte(options.GetString(Keys.Cert)),
			[]byte(options.GetString(Keys.Key)),
		)
	}

	return tls.LoadX509KeyPair(options.GetString(Keys.Cert), options.GetString(Keys.Key))
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
	options.SetDefault(Keys.CertsFromEnv, false)
	options.SetDefault(Keys.Port, "3000")
	options.SetDefault(Keys.SubsHost, "https://subscription.api.redhat.com")
	options.SetDefault(Keys.CaPath, "../resources/ca.crt")
	options.SetDefault(Keys.Cert, "../test_data/test.cert")
	options.SetDefault(Keys.Key, "../test_data/test.key")
	options.SetDefault(Keys.OpenAPISpecPath, "./apispec/api.spec.json")
	options.SetDefault(Keys.BundleInfoYaml, "./bundles/bundles.yml")
	options.SetEnvPrefix("ENT")
	options.AutomaticEnv()

	config = &EntitlementsConfig{
		Certs:   getCerts(options),
		RootCAs: getRootCAs(options.GetString(Keys.CaPath)),
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
