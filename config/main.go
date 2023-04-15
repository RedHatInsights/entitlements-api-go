package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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
	LogLevel        string
	CertsFromEnv    string
	SubsHost        string
	ComplianceHost  string
	CaPath          string
	OpenAPISpecPath string
	BundleInfoYaml  string
	CwLogGroup      string
	CwLogStream     string
	CwRegion        string
	CwKey           string
	CwSecret        string
	Features        string
	FeaturesPath    string
	SubAPIBasePath  string
	CompAPIBasePath string
	RunBundleSync   string
	EntitleAll      string
	AMSHost         string
	ClientID        string
	ClientSecret    string
	TokenURL        string
	Debug           string
	BOPClientID     string
	BOPToken        string
	BOPURL          string
	BOPMockResponse string
}

// Keys is a struct that houses all the env variables key names
var Keys = EntitlementsConfigKeysType{
	Key:             "KEY",
	Cert:            "CERT",
	Port:            "PORT",
	LogLevel:        "LOG_LEVEL",
	CertsFromEnv:    "CERTS_FROM_ENV",
	SubsHost:        "SUBS_HOST",
	ComplianceHost:  "COMPLIANCE_HOST",
	CaPath:          "CA_PATH",
	OpenAPISpecPath: "OPENAPI_SPEC_PATH",
	BundleInfoYaml:  "BUNDLE_INFO_YAML",
	CwLogGroup:      "CW_LOG_GROUP",
	CwLogStream:     "CW_LOG_STEAM",
	CwRegion:        "CW_REGION",
	CwKey:           "CW_KEY",
	CwSecret:        "CW_SECRET",
	Features:        "FEATURES",
	SubAPIBasePath:  "SUB_API_BASE_PATH",
	CompAPIBasePath: "COMP_API_BASE_PATH",
	RunBundleSync:   "RUN_BUNDLE_SYNC",
	EntitleAll:      "ENTITLE_ALL",
	AMSHost:         "AMS_HOST",
	ClientID:        "OIDC_CLIENT_ID",
	ClientSecret:    "OIDC_CLIENT_SECRET",
	TokenURL:        "OAUTH_TOKEN_URL",
	BOPClientID:     "BOP_CLIENT_ID",
	BOPToken:        "BOP_TOKEN",
	BOPURL:          "BOP_URL",
	BOPMockResponse: "BOP_MOCK_RESPONSE",
	Debug:           "DEBUG",
}

func getBaseFeaturesPath(options *viper.Viper) string {
	featureList := strings.Split(options.GetString(Keys.Features), ",")
	return "?features=" + strings.Join(featureList, "&features=")
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
	hostname, err := os.Hostname()

	if err != nil {
		hostname = "entitlements"
	}

	options.SetDefault(Keys.CertsFromEnv, false)
	options.SetDefault(Keys.Port, "3000")
	options.SetDefault(Keys.LogLevel, "info")
	options.SetDefault(Keys.SubsHost, "https://subscription.api.redhat.com")
	options.SetDefault(Keys.ComplianceHost, "https://export-compliance.api.redhat.com")
	options.SetDefault(Keys.CaPath, "../resources/ca.crt")
	options.SetDefault(Keys.Cert, "../test_data/test.cert") // default values of Cert and Key are for testing purposes only
	options.SetDefault(Keys.Key, "../test_data/test.key")
	options.SetDefault(Keys.OpenAPISpecPath, "./apispec/api.spec.json")
	options.SetDefault(Keys.BundleInfoYaml, "./bundles/bundles.yml")
	options.SetDefault(Keys.CwLogGroup, "platform-dev")
	options.SetDefault(Keys.CwLogStream, hostname)
	options.SetDefault(Keys.CwRegion, "us-east-1")
	options.SetDefault(Keys.Features, "ansible,smart_management,rhods,rhoam,rhosak,openshift")
	options.SetDefault(Keys.SubAPIBasePath, "/svcrest/subscription/v5/")
	options.SetDefault(Keys.CompAPIBasePath, "/v1/screening")
	options.SetDefault(Keys.RunBundleSync, false)
	options.SetDefault(Keys.EntitleAll, false)
	options.SetDefault(Keys.AMSHost, "https://api.openshift.com")
	options.SetDefault(Keys.TokenURL, "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token")
	options.SetDefault(Keys.BOPURL, "https://backoffice-proxy.apps.ext.spoke.prod.us-west-2.aws.paas.redhat.com/v1/users")
	options.SetDefault(Keys.BOPMockResponse, `{"orgId":"4384938490324", "code":200, "users":"[{\"org_id\":\"4384938490324\", \"username\":\"foo\"}]"}`)
	options.SetDefault(Keys.Debug, false)

	options.SetEnvPrefix("ENT")
	options.AutomaticEnv()

	// Must be set after AutomaticEnv() in order to pickup FEATURES env variable
	options.SetDefault(Keys.FeaturesPath, getBaseFeaturesPath(options))

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
