package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/ams"
	"github.com/RedHatInsights/entitlements-api-go/api"
	"github.com/RedHatInsights/entitlements-api-go/bop"
	"github.com/RedHatInsights/entitlements-api-go/config"
	ocmErrors "github.com/openshift-online/ocm-sdk-go/errors"
)

// There are errors from AMS that we want to add some contextual info to before bubbling up to our clients.
// This mapper provides a way to record known AMS error codes, and map error messages to them via env variables.

// AMS Error Codes
const acctMgmt11 	= "ACCT-MGMT-11"
const acctMgmt11Status 	= http.StatusForbidden

type SeatsErrorMapper interface {
	MapResponse(err error, httpStatusCode int) api.Error
}

type DefaultMapper struct {
	config *config.EntitlementsConfig
}

func NewErrorMapper(config *config.EntitlementsConfig) SeatsErrorMapper {
	return &DefaultMapper{
		config: config,
	}
}

func(m *DefaultMapper) MapResponse(err error, httpStatusCode int) api.Error {
	var amsError *ocmErrors.Error
	if errors.As(err, &amsError) {
		reason := m.mapAMSErrorMessage(amsError)
		return api.Error{
			Error: 			toPtr(reason),
			Code:  			toPtr(amsError.Code()),
			Identifier: 	toPtr(amsError.ID()),
			OperationId: 	toPtr(amsError.OperationID()),
			Status: 		toPtr(amsError.Status()),
		}
	} 
	
	var clientError *ams.ClientError
	if errors.As(err, &clientError) {
		return api.Error{
			Error: 	toPtr(clientError.Error()),
			Status: toPtr(clientError.StatusCode),
		}
	} 

	var userDetailErr *bop.UserDetailError
	if errors.As(err, &userDetailErr) {
		return api.Error{
			Error: 	toPtr(userDetailErr.Error()),
			Status: toPtr(userDetailErr.StatusCode),
		}
	}

	return api.Error{
		Error: toPtr(err.Error()),
		Status: toPtr(httpStatusCode),
	}
}

func(m *DefaultMapper) mapAMSErrorMessage(err *ocmErrors.Error) string {
	if isError(err, acctMgmt11, acctMgmt11Status) {
		return fmt.Sprintf("%s. %s", err.Reason(), m.config.Options.GetString(config.Keys.AMSAcctMgmt11Msg))
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
