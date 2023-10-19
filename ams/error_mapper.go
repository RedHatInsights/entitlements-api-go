package ams

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
	ocmErrors "github.com/openshift-online/ocm-sdk-go/errors"
)

// There are errors from AMS that we want to add some contextual info to before bubbling up to our clients.
// This mapper provides a way to record known AMS error codes, and map error messages to them via env variables.

// AMS Error Codes
const acctMgmt11 		= "ACCT-MGMT-11"
const acctMgmt11Status 	= http.StatusForbidden

type AMSErrorMapper interface {
	MapErrorMessage(err *ocmErrors.Error) string
}

type DefaultMapper struct {
	config *config.EntitlementsConfig
}

func(m *DefaultMapper) MapErrorMessage(err *ocmErrors.Error) string {
	if isError(err, acctMgmt11, acctMgmt11Status) {
		return fmt.Sprintf("%s. %s.", err.Reason(), m.config.Options.GetString(config.Keys.AMSAcctMgmt11Msg))
	}

	return err.Reason()
}

func isError(err *ocmErrors.Error, errorCode string, status int) bool {
	return err.Code() == errorCode && err.Status() == status
}

type MockMapper struct {
	Mock func(err *ocmErrors.Error) string
}

func(m *MockMapper) MapErrorMessage(err *ocmErrors.Error) string {
	return m.Mock(err)
}

func NewErrorMapper(config *config.EntitlementsConfig) AMSErrorMapper {
	return &DefaultMapper{
		config: config,
	}
}