package controllers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/redhatinsights/platform-go-middlewares/identity"

	"github.com/getsentry/sentry-go"
	"github.com/karlseguin/ccache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type getter func(string) []string

var cache = ccache.New(ccache.Configure().MaxSize(500).ItemsToPrune(50))
var bundleInfo []types.Bundle
var subsFailure = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "it_subscriptions_service_failure",
		Help: "Total number of IT subscriptions service failures",
	},
	[]string{"code"},
)
var subsTimeHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "it_subscriptions_service_time_taken",
	Help:    "Subscriptions latency distributions.",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})

// SetBundleInfo sets the bundle information fetched from the YAML
func SetBundleInfo(yamlFilePath string) error {
	bundlesYaml, err := ioutil.ReadFile(yamlFilePath)

	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	err = yaml.Unmarshal([]byte(bundlesYaml), &bundleInfo)
	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	return nil
}

// GetFeatureStatus calls the IT subs service features endpoint and returns the entitlements for specified features/bundles
var GetFeatureStatus = func(orgID string) types.SubscriptionsResponse {
	item := cache.Get(orgID)

	if item != nil && !item.Expired() {
		return types.SubscriptionsResponse{
			StatusCode: 200,
			Data:       item.Value().(types.FeatureStatus),
			CacheHit:   true,
		}
	}

	q := config.GetConfig().Options.GetString(config.Keys.FeaturesPath)
	req := config.GetConfig().Options.GetString(config.Keys.SubsHost) +
		"/svcrest/subscription/v5/featureStatus" +
		q + "&accountId=" + orgID

	resp, err := getClient().Get(req)

	if err != nil {
		sentry.CaptureException(err)
		return types.SubscriptionsResponse{
			Error: err,
		}
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		return types.SubscriptionsResponse{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			Error:      nil,
			Data:       types.FeatureStatus{},
			CacheHit:   false,
		}
	}

	defer resp.Body.Close()

	// Unmarshaling response from Feature service
	body, _ := ioutil.ReadAll(resp.Body)
	var FeatureStatus types.FeatureStatus
	json.Unmarshal(body, &FeatureStatus)

	cache.Set(orgID, FeatureStatus, time.Minute*10)

	return types.SubscriptionsResponse{
		StatusCode: resp.StatusCode,
		Data:       FeatureStatus,
		CacheHit:   false,
	}
}

func failOnDependencyError(errMsg string, res types.SubscriptionsResponse, w http.ResponseWriter) {
	dependencyError := types.DependencyErrorDetails{
		DependencyFailure: true,
		Service:           "Subscriptions Service",
		Status:            res.StatusCode,
		Endpoint:          config.GetConfig().Options.GetString(config.Keys.SubsHost),
		Message:           errMsg,
	}

	errorResponse := types.DependencyErrorResponse{Error: dependencyError}
	errorResponsejson, _ := json.Marshal(errorResponse)

	subsFailure.WithLabelValues(strconv.Itoa(res.StatusCode)).Inc()
	http.Error(w, string(errorResponsejson), 500)
}

// Index the handler for GETs to /api/entitlements/v1/services/
func Index() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var include_filter []string
		var exclude_filter []string
		start := time.Now()
		idObj := identity.Get(req.Context()).Identity
		orgId := idObj.Internal.OrgID
		res := GetFeatureStatus(orgId)
		accNum := idObj.AccountNumber
		isInternal := idObj.User.Internal
		validEmailMatch, _ := regexp.MatchString(`^.*@redhat.com$`, idObj.User.Email)

		validAccNum := !(accNum == "" || accNum == "-1")
		validOrgId := !(orgId == "" || orgId == "-1")

		include_list := req.URL.Query().Get("include_bundles")
		if include_list != "" {
			include_filter = strings.Split(include_list, ",")
		} else {
			exclude_list := req.URL.Query().Get("exclude_bundles")
			if exclude_list != "" {
				exclude_filter = strings.Split(exclude_list, ",")
			}
		}

		if res.Error != nil {
			errMsg := "Unexpected error while talking to Subs Service"
			l.Log.WithFields(logrus.Fields{"error": res.Error}).Error(errMsg)
			sentry.CaptureException(res.Error)
			failOnDependencyError(errMsg, res, w)
			return
		}

		subsTimeTaken := time.Since(start).Seconds()
		l.Log.WithFields(logrus.Fields{"subs_call_duration": subsTimeTaken, "cache_hit": res.CacheHit}).Info("subs call complete")
		subsTimeHistogram.Observe(subsTimeTaken)

		if res.StatusCode != 200 {
			errMsg := "Got back a non 200 status code from Subscriptions Service"
			l.Log.WithFields(logrus.Fields{"code": res.StatusCode, "body": res.Body}).Error(errMsg)

			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetExtra("response_body", res.Body)
				scope.SetExtra("response_status", res.StatusCode)
				sentry.CaptureException(errors.New(errMsg))
			})

			failOnDependencyError(errMsg, res, w)
			return
		}

		entitlementsResponse := make(map[string]types.EntitlementsSection)
		for _, b := range bundleInfo {
			if len(include_filter) > 0 && !contains(include_filter, b.Name) {
				continue
			} else if len(exclude_filter) > 0 && contains(exclude_filter, b.Name) {
				continue
			}

			entitle := true
			trial := false

			if len(b.Skus) > 0 {
				entitle = false
				for _, f := range res.Data.Features {
					if f.Name == b.Name {
						entitle = f.Entitled
						trial = f.IsEval
					}
				}
			}

			if b.UseValidAccNum {
				entitle = validAccNum && entitle
			}

			if b.UseValidOrgId {
				entitle = validOrgId && entitle
			}

			if b.UseIsInternal {
				entitle = validAccNum && isInternal && validEmailMatch
			}
			entitlementsResponse[b.Name] = types.EntitlementsSection{IsEntitled: entitle, IsTrial: trial}
		}

		obj, err := json.Marshal(entitlementsResponse)

		if err != nil {
			l.Log.WithFields(logrus.Fields{"error": err}).Error("Unexpected error while unmarshalling JSON data from Subs Service")
			sentry.CaptureException(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(obj))
	}
}

func contains(s []string, t string) bool {
	for _, val := range s {
		if val == t {
			return true
		}
	}
	return false
}
