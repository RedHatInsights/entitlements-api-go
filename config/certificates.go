package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// The path for the CA file defined in the "it-key-pair" secret and the "it-certificates" volume mount.
	CaCertiticateFilePath = "/certificates/ca.crt"
	// The path for the certificate file defined in the "it-key-pair" secret and the "it-certificates" volume mount.
	CertificateFilePath = "/certificates/tls.crt"
	// The path for the key file defined in the "it-key-pair" secret and the "it-certificates" volume mount.
	KeyFilePath = "/certificates/tls.key"
)

// loadCertificates loads the certificate-key pair and the custom CA
// certificate into the given configuration.
func loadCertificates(config *EntitlementsConfig) error {
	// Load the certificates from the environment variables.
	if config.Options.GetBool(Keys.CertsFromEnv) {
		err := loadCertificatesFromEnvironmentVariables(config)
		if err != nil {
			return fmt.Errorf("unable to load the certificates from the environment: %w", err)
		}

		log.Println("Certificates loaded from the environment variables")
		return nil
	}

	// Set the default file paths for the certificates.
	workingDirectory := getWorkingDirectory()
	caCertFilePath := fmt.Sprintf("%s/../test_data/test_ca_chain.cert", workingDirectory)
	certFilePath := fmt.Sprintf("%s/../test_data/test.cert", workingDirectory)
	keyFilePath := fmt.Sprintf("%s/../test_data/test.key", workingDirectory)

	// When the "it-certificates" volume mount is present, we read the
	// certificates from there.
	if isCertificatesVolumeMounted() {
		log.Println("Reading certificates from the volume mount")

		caCertFilePath = CaCertiticateFilePath
		certFilePath = CertificateFilePath
		keyFilePath = KeyFilePath
	} else {
		log.Println("Reading the certificates from the test directory")
	}

	err := loadCertificateKeyPairFromFile(config, certFilePath, keyFilePath)
	if err != nil {
		return fmt.Errorf(`unable to load certificate-key pair: %w`, err)
	}

	err = loadCAFileIntoSystemCAsBundle(config, caCertFilePath)
	if err != nil {
		return fmt.Errorf(`unable to load the CA certificate: %w`, err)
	}

	return nil
}

// getWorkingDirectory returns the currently working directory, or sets it to
// the default current directory ".".
func getWorkingDirectory() string {
	wd := "."
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Printf("Error getting runtime caller, some default config settings might not be set correctly. Working directory set to: [%s]\n", wd)
	} else {
		wd = filepath.Dir(filename)
	}

	return wd
}

// isCertificatesVolumeMounted checks whether the "it-certificates" volume is
// mounted.
func isCertificatesVolumeMounted() bool {
	_, err := os.Stat("/certificates")

	return !os.IsNotExist(err)
}

// getSysmteCAs gets the system's certificate pool to be able to add
// certificates to it if needed.
func getSystemCAs() (*x509.CertPool, error) {
	rootCAs, err := x509.SystemCertPool()
	if rootCAs == nil {
		return nil, fmt.Errorf("unable to load system CA certificates")
	}

	if err != nil {
		return nil, fmt.Errorf("unable to load system CA certificates: %w", err)
	}

	return rootCAs, nil
}

// loadCAFileIntoSystemCAsBundle loads the certificate file from the given path
// and appends it to the system's CA bundle.
func loadCAFileIntoSystemCAsBundle(config *EntitlementsConfig, caFilePath string) error {
	rootCAs, err := getSystemCAs()
	if err != nil {
		return fmt.Errorf("unable to get system CAs: %w", err)
	}

	certs, err := os.ReadFile(caFilePath)
	if err != nil {
		return fmt.Errorf(`unable to load CA file "%s" to append it to the RootCAs: %w`, caFilePath, err)
	}

	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		return fmt.Errorf("failed to AppendCertsFromPEM %s to RootCAs", caFilePath)
	}

	config.RootCAs = rootCAs

	return nil
}

// loadCertificatesFromEnvironmentVariables loads the certificates from the
// "ENT_CA_CERT", "ENT_CERT" and "ENT_KEY" environment variables into the
// application's configuration.
func loadCertificatesFromEnvironmentVariables(config *EntitlementsConfig) error {
	// Load the certificate-key pair.
	certificatePair, err := tls.X509KeyPair(
		[]byte(config.Options.GetString(Keys.Cert)),
		[]byte(config.Options.GetString(Keys.Key)),
	)

	if err != nil {
		return fmt.Errorf("unable to read the certificates from the environment variables: %w", err)
	}

	config.Certs = &certificatePair

	// Load the CA cert by appending it to the system's certificates.
	systemCAs, err := getSystemCAs()
	if err != nil {
		return fmt.Errorf("unable to get the system CAs: %w", err)
	}

	ok := systemCAs.AppendCertsFromPEM([]byte(config.Options.GetString(Keys.CaCert)))
	if !ok {
		return fmt.Errorf("CA certificate not appended to the system CAs")
	}

	config.RootCAs = systemCAs

	return nil
}

// loadCertificateKeyPairFromFile loads the certificate-key pair from the given
// file paths into the application's configuration.
func loadCertificateKeyPairFromFile(config *EntitlementsConfig, certificateFilePath, keyFilePath string) error {
	tlsCertificate, err := tls.LoadX509KeyPair(certificateFilePath, keyFilePath)
	if err != nil {
		return fmt.Errorf(`unable to load the certificate-key pairs from files "%s" and "%s": %s`, certificateFilePath, keyFilePath, err)
	}

	config.Certs = &tlsCertificate
	return nil
}
