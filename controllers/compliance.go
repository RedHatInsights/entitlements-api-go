package controllers

import (
	"bytes"
	"encoding/json"
	"github.com/RedHatInsights/entitlements-api-go/config"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var screeningPathV1 = "/v1/screening"
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

		complianceTimeTaken := time.Since(start).Seconds()
		l.Log.WithFields(logrus.Fields{"compliance_call_duration": complianceTimeTaken}).Info("compliance call complete")
		complianceTimeHistogram.Observe(complianceTimeTaken)

		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(respBody))
	}
}

func failOnBadRequest(w http.ResponseWriter, errMsg string, err error) {
	sentry.CaptureException(err)
	l.Log.WithFields(logrus.Fields{"error": err}).Error(errMsg)
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

func failOnComplianceError(w http.ResponseWriter, errMsg string, err error, url string) {
	sentry.CaptureException(err)
	l.Log.WithFields(logrus.Fields{"error": err}).Error(errMsg)
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
