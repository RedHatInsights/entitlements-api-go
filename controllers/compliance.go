package controllers

import (
	"bytes"
	"encoding/json"
	"github.com/RedHatInsights/entitlements-api-go/config"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
)

var screeningPathV1 = "/v1/screening"
var complianceServiceName = "Export Compliance Service"

func Compliance() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		// TODO: time metrics
		reqBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			failOnBadRequest(w, "Failed to read request body", err)
			return
		}
		defer req.Body.Close()
		var httpClient = getClient()

		url := config.GetConfig().Options.GetString(config.Keys.ComplianceHost) + screeningPathV1
		complianceReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))

		if err != nil {
			failOnComplianceError(w, "Unexpected error while creating request to Export Compliance Service", err, url)
			return
		}

		complianceReq.Header.Add("accept", "application/json;charset=UTF-8")
		complianceReq.Header.Add("Content-Type", "application/json;charset=UTF-8")

		resp, err := httpClient.Do(complianceReq)
		if err != nil {
			failOnComplianceError(w, "Unexpected error returned on request to Export Compliance Service", err, url)
			return
		}

		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(respBody))
	}
}

func failOnBadRequest(w http.ResponseWriter, errMsg string, err error) {
	sentry.CaptureException(err)
	l.Log.WithFields(logrus.Fields{"error": err}).Error(errMsg)

	response := types.RequestErrorResponse{
		Error: types.RequestErrorDetails{
			Status:  http.StatusBadRequest,
			Message: errMsg + ": " + err.Error(),
		},
	}

	responseJson, _ := json.Marshal(response)
	http.Error(w, string(responseJson), http.StatusBadRequest)
}

func failOnComplianceError(w http.ResponseWriter, errMsg string, err error, url string) {
	sentry.CaptureException(err)
	l.Log.WithFields(logrus.Fields{"error": err}).Error(errMsg)
	response := types.DependencyErrorResponse{
		Error: types.DependencyErrorDetails{
			DependencyFailure: true,
			Service:           complianceServiceName,
			Status:            http.StatusInternalServerError,
			Endpoint:          url,
			Message:           err.Error(),
		},
	}

	responseJson, _ := json.Marshal(response)
	http.Error(w, string(responseJson), http.StatusInternalServerError)
}
