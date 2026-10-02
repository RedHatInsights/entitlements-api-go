package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	u "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/securitylog"
	"github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"github.com/sirupsen/logrus"
)

var complianceServiceName = "Export Compliance Service"

var complianceFailure = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "it_export_compliance_service_failure",
		Help: "Total number of IT export compliance service failures",
	},
	[]string{"code"},
)
var complianceTimeHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "it_export_compliance_service_time_taken",
	Help:    "Export compliance service latency distributions.",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})

func Compliance() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()

		userIdentity := identity.GetIdentity(req.Context()).Identity
		resourceID := securitylog.ResourceIDOrFallback(userIdentity.AccountNumber, "compliance-screening")
		if userIdentity.User != nil && strings.TrimSpace(userIdentity.User.Username) != "" {
			resourceID = userIdentity.User.Username
		}

		// Service Accounts don't have User field and cannot be screened for compliance
		if userIdentity.User == nil {
			err := errors.New("compliance: Service Accounts are not supported for compliance screening")
			failOnBadRequest(w, userIdentity, resourceID, "Invalid identity type", err)
			return
		}

		if len(strings.TrimSpace(userIdentity.User.Username)) == 0 {
			err := errors.New("compliance: x-rh-identity header has a missing or whitespace username")
			failOnBadRequest(w, userIdentity, resourceID, "Invalid x-rh-identity header", err)
			return
		}

		reqBody := constructComplianceRequestBody(userIdentity)

		reqBodyJson, err := json.Marshal(reqBody)
		if err != nil {
			failOnServiceError(w, userIdentity, resourceID, "Unable to marshal request to compliance service", err)
			return
		}

		var httpClient = getClient()
		configOptions := config.GetConfig().Options
		url := configOptions.GetString(config.Keys.ComplianceHost) + configOptions.GetString(config.Keys.CompAPIBasePath)
		complianceReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBodyJson))

		if err != nil {
			failOnComplianceError(w, userIdentity, resourceID, "Unexpected error while creating request to Export Compliance Service", err, url)
			return
		}

		complianceReq.Header.Add("accept", "application/json;charset=UTF-8")
		complianceReq.Header.Add("Content-Type", "application/json;charset=UTF-8")

		resp, err := httpClient.Do(complianceReq)
		if err != nil {
			var urlError *u.Error
			if errors.As(err, &urlError) && urlError.Timeout() {
				failOnComplianceError(w, userIdentity, resourceID, "Request to Export Compliance Service timed out", err, url)
			} else {
				failOnComplianceError(w, userIdentity, resourceID, "Unexpected error returned on request to Export Compliance Service", err, url)
			}

			return
		}

		complianceTimeTaken := time.Since(start).Seconds()
		l.Log.WithFields(logrus.Fields{"compliance_call_duration": complianceTimeTaken}).Info("compliance call complete")
		complianceTimeHistogram.Observe(complianceTimeTaken)

		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write([]byte(respBody))
		if resp.StatusCode >= http.StatusBadRequest {
			// READ compliance failure - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation, EOI-7 invalid_login, EOI-11 warnings_or_errors)
			l.Log.WithFields(logrus.Fields{"status": resp.StatusCode}).WithFields(securitylog.FieldsFromIdentity(
				userIdentity,
				"READ",
				"export_compliance_screening",
				resourceID,
				securitylog.OutcomeFailure,
			)).Warn("Compliance screening returned non-success status")
			return
		}

		// READ compliance success - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation)
		l.Log.WithFields(securitylog.FieldsFromIdentity(
			userIdentity,
			"READ",
			"export_compliance_screening",
			resourceID,
			securitylog.OutcomeSuccess,
		)).Info("Compliance screening succeeded")
	}
}

func constructComplianceRequestBody(userIdentity identity.Identity) types.ComplianceScreeningRequest {
	reqBody := types.ComplianceScreeningRequest{
		User: types.User{
			Login: userIdentity.User.Username,
		},
		Account: types.Account{
			Primary: true,
		},
	}

	return reqBody
}

func failOnBadRequest(w http.ResponseWriter, idObj identity.Identity, resourceID string, errMsg string, err error) {
	sentry.CaptureException(err)
	// READ compliance validation failure - SEC-MON-REQ-1 compliance (EOI-7 invalid_login, EOI-11 warnings_or_errors)
	l.Log.WithFields(logrus.Fields{"error": err}).WithFields(securitylog.FieldsFromIdentity(
		idObj,
		"READ",
		"export_compliance_screening",
		resourceID,
		securitylog.OutcomeFailure,
	)).Warn(errMsg)
	complianceFailure.WithLabelValues(strconv.Itoa(http.StatusBadRequest)).Inc()

	response := types.RequestErrorResponse{
		Error: types.RequestErrorDetails{
			Status:  http.StatusBadRequest,
			Message: errMsg + ": " + err.Error(),
		},
	}

	responseJson, _ := json.Marshal(response)
	http.Error(w, string(responseJson), http.StatusBadRequest)
}

func failOnComplianceError(w http.ResponseWriter, idObj identity.Identity, resourceID, errMsg string, err error, url string) {
	sentry.CaptureException(err)
	// READ compliance dependency failure - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation, EOI-11 warnings_or_errors)
	l.Log.WithFields(logrus.Fields{"error": err, "url": url}).WithFields(securitylog.FieldsFromIdentity(
		idObj,
		"READ",
		"export_compliance_screening",
		resourceID,
		securitylog.OutcomeFailure,
	)).Error(errMsg)
	complianceFailure.WithLabelValues(strconv.Itoa(http.StatusInternalServerError)).Inc()

	response := types.DependencyErrorResponse{
		Error: types.DependencyErrorDetails{
			DependencyFailure: true,
			Service:           complianceServiceName,
			Status:            http.StatusInternalServerError,
			Endpoint:          url,
			Message:           errMsg + ": " + err.Error(),
		},
	}

	responseJson, _ := json.Marshal(response)
	http.Error(w, string(responseJson), http.StatusInternalServerError)
}

func failOnServiceError(w http.ResponseWriter, idObj identity.Identity, resourceID, errMsg string, err error) {
	// READ compliance service failure - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation, EOI-11 warnings_or_errors)
	l.Log.WithFields(logrus.Fields{"error": err}).WithFields(securitylog.FieldsFromIdentity(
		idObj,
		"READ",
		"export_compliance_screening",
		resourceID,
		securitylog.OutcomeFailure,
	)).Error(errMsg)
	sentry.CaptureException(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError)+": "+errMsg+": "+err.Error(), http.StatusInternalServerError)
}
