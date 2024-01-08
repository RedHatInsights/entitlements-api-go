package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
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
	Key                string
	Cert               string
	Port               string
	LogLevel           string
	CertsFromEnv       string
	SubsHost           string
	ComplianceHost     string
	CaPath             string
	OpenAPISpecPath    string
	BundleInfoYaml     string
	CwLogGroup         string
	CwLogStream        string
	CwRegion           string
	CwKey              string
	CwSecret           string
	Features           string
	SubAPIBasePath     string
	CompAPIBasePath    string
	RunBundleSync      string
	EntitleAll         string
	AMSHost            string
	ClientID           string
	ClientSecret       string
	TokenURL           string
	Debug              string
	BOPClientID        string
	BOPToken           string
	BOPURL             string
	BOPEnv             string
	BOPMockOrgId       string
	DisableSeatManager string
	SubsCacheDuration  string
	SubsCacheMaxSize   string
	SubsCacheItemPrune string
	AMSAcctMgmt11Msg   string
}

// Keys is a struct that houses all the env variables key names
var Keys = EntitlementsConfigKeysType{
	Key:                "KEY",
	Cert:               "CERT",
	Port:               "PORT",
	LogLevel:           "LOG_LEVEL",
	CertsFromEnv:       "CERTS_FROM_ENV",
	SubsHost:           "SUBS_HOST",
	ComplianceHost:     "COMPLIANCE_HOST",
	CaPath:             "CA_PATH",
	OpenAPISpecPath:    "OPENAPI_SPEC_PATH",
	BundleInfoYaml:     "BUNDLE_INFO_YAML",
	CwLogGroup:         "CW_LOG_GROUP",
	CwLogStream:        "CW_LOG_STEAM",
	CwRegion:           "CW_REGION",
	CwKey:              "CW_KEY",
	CwSecret:           "CW_SECRET",
	Features:           "FEATURES",
	SubAPIBasePath:     "SUB_API_BASE_PATH",
	CompAPIBasePath:    "COMP_API_BASE_PATH",
	RunBundleSync:      "RUN_BUNDLE_SYNC",
	EntitleAll:         "ENTITLE_ALL",
	AMSHost:            "AMS_HOST",
	ClientID:           "OIDC_CLIENT_ID",
	ClientSecret:       "OIDC_CLIENT_SECRET",
	TokenURL:           "OAUTH_TOKEN_URL",
	BOPClientID:        "BOP_CLIENT_ID",
	BOPToken:           "BOP_TOKEN",
	BOPURL:             "BOP_URL",
	BOPMockOrgId:       "BOP_MOCK_ORG_ID",
	BOPEnv:             "BOP_ENV",
	Debug:              "DEBUG",
	DisableSeatManager: "DISABLE_SEAT_MANAGER",
	SubsCacheDuration:  "SUBS_CACHE_DURATION_SECONDS",
	SubsCacheMaxSize:   "SUBS_CACHE_MAX_SIZE",
	SubsCacheItemPrune: "SUBS_CACHE_ITEM_PRUNE",
	AMSAcctMgmt11Msg:	"AMS_ACCT_MGMT_11_ERR_MSG",
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

	certs, err := os.ReadFile(localCertFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to append %q to RootCAs: %v", localCertFile, err))
	}

	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		panic(fmt.Sprintf("Failed to AppendCertsFromPEM %q to RootCAs", localCertFile))
	}

	return rootCAs
}

func loadCerts(options *viper.Viper) (tls.Certificate, error) {
	if options.GetBool(Keys.CertsFromEnv) {
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

	wd := "."
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Printf("Error getting runtime caller, some default config settings might not be set correctly. Working directory set to: [%s]\n", wd)
	} else {
		wd = filepath.Dir(filename)
	}

	options.SetDefault(Keys.CertsFromEnv, false)
	options.SetDefault(Keys.Port, "3000")
	options.SetDefault(Keys.LogLevel, "info")
	options.SetDefault(Keys.SubsHost, "https://subscription.api.redhat.com")
	options.SetDefault(Keys.ComplianceHost, "https://export-compliance.api.redhat.com")
	options.SetDefault(Keys.CaPath, fmt.Sprintf("%s/../resources/ca.crt", wd))
	options.SetDefault(Keys.Cert, fmt.Sprintf("%s/../test_data/test.cert", wd)) // default values of Cert and Key are for testing purposes only
	options.SetDefault(Keys.Key, fmt.Sprintf("%s/../test_data/test.key", wd))
	options.SetDefault(Keys.OpenAPISpecPath, "./apispec/api.spec.json")
	options.SetDefault(Keys.BundleInfoYaml, "./bundles/bundles.yml")
	options.SetDefault(Keys.CwLogGroup, "platform-dev")
	options.SetDefault(Keys.CwLogStream, hostname)
	options.SetDefault(Keys.CwRegion, "us-east-1")
	options.SetDefault(Keys.SubAPIBasePath, "/svcrest/subscription/v5/")
	options.SetDefault(Keys.CompAPIBasePath, "/v1/screening")
	options.SetDefault(Keys.RunBundleSync, false)
	options.SetDefault(Keys.EntitleAll, false)
	options.SetDefault(Keys.AMSHost, "https://api.openshift.com")
	options.SetDefault(Keys.TokenURL, "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token")
	options.SetDefault(Keys.BOPURL, "https://backoffice-proxy.apps.ext.spoke.prod.us-west-2.aws.paas.redhat.com/v1/users")
	options.SetDefault(Keys.BOPMockOrgId, "4384938490324")
	options.SetDefault(Keys.BOPEnv, "stage")
	options.SetDefault(Keys.Debug, false)
	options.SetDefault(Keys.DisableSeatManager, false)
	options.SetDefault(Keys.SubsCacheDuration, 1800) // seconds
	options.SetDefault(Keys.SubsCacheMaxSize, 500)
	options.SetDefault(Keys.SubsCacheItemPrune, 50)
	options.SetDefault(Keys.AMSAcctMgmt11Msg, "Please have this user log into \"https://console.redhat.com/openshift\" to grant their account the required permissions, or try again later.")

	options.SetEnvPrefix("ENT")
	options.AutomaticEnv()

	config = &EntitlementsConfig{
		Certs:   getCerts(options),
		RootCAs: getRootCAs(options.GetString(Keys.CaPath)),
		Options: options,
	}

	if clowder.IsClowderEnabled() {
		cfg := clowder.LoadedConfig

		// Cloudwatch
		options.Set(Keys.CwLogGroup, cfg.Logging.Cloudwatch.LogGroup)
		options.Set(Keys.CwLogStream, cfg.Logging.Cloudwatch.LogGroup)
		options.Set(Keys.CwRegion, cfg.Logging.Cloudwatch.Region)
		options.Set(Keys.CwKey, cfg.Logging.Cloudwatch.AccessKeyId)
		options.Set(Keys.CwSecret, cfg.Logging.Cloudwatch.SecretAccessKey)
	}
}

// GetConfig provides a singleton global EntitlementsConfig instance
func GetConfig() *EntitlementsConfig {
	if config == nil {
		initialize()
	}

	return config
}
