package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"

	"github.com/getsentry/sentry-go"
	"github.com/karlseguin/ccache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var configOptions = config.GetConfig().Options
var cache = ccache.New(
	ccache.Configure[types.FeatureStatus]().
		MaxSize(configOptions.GetInt64(config.Keys.SubsCacheMaxSize)).
		PercentToPrune(uint8(configOptions.GetUint32(config.Keys.SubsCacheItemPrune))),
)
var cacheDuration = time.Second * time.Duration(configOptions.GetInt64(config.Keys.SubsCacheDuration))

var bundleInfo []types.Bundle
var featuresQuery string
var paidFeatureSuffix string

// paidFeatures maps a SKU feature name to whether it has a "_paid" variant in the Feature
// Service catalog. It is populated once at startup by InitPaidFeatures (from /features/v1)
// and used only to decide is_trial. A nil map (e.g. in tests that don't init it) reads as
// all-false, so is_trial safely defaults to false.
var paidFeatures map[string]bool
var subsFailure = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "it_feature_service_failure",
		Help: "Total number of IT feature service failures",
	},
	[]string{"code"},
)
var subsTimeHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "it_feature_service_time_taken",
	Help:    "Feature service latency distributions.",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})

type GetServicesParams struct {
	IncludeBundles []string
	ExcludeBundles []string
	TrialActivated bool
}

const (
	IncludeBundlesParamKey string = "include_bundles"
	ExcludeBundlesParamKey string = "exclude_bundles"
	TrialActivatedParamKey string = "trial_activated"
)

type GetFeatureStatusParams struct {
	OrgId          string
	ForceFreshData bool
}

// SetBundleInfo sets the bundle information fetched from the YAML
func SetBundleInfo(yamlFilePath string) error {
	bundlesYaml, err := os.ReadFile(yamlFilePath)

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

// nonSkuBundleNames returns the set of bundle names in bundles.yml that are NOT SKU-based.
// These are still sourced from bundles.yml (acc num / org id / internal gating).
func nonSkuBundleNames() map[string]bool {
	names := make(map[string]bool)
	for _, bundle := range bundleInfo {
		if !bundle.IsSkuBased() {
			names[bundle.Name] = true
		}
	}
	return names
}

// skuFeatures returns the SKU-based features to resolve against Feature Service: every
// entry in ENT_FEATURES, minus any name that is a non-SKU bundle in bundles.yml. The
// subtraction keeps non-SKU bundles (e.g. openshift) out of the SKU path even if they are
// still listed in ENT_FEATURES during the config transition.
func skuFeatures() []string {
	nonSku := nonSkuBundleNames()
	var features []string
	for _, f := range strings.Split(configOptions.GetString(config.Keys.Features), ",") {
		f = strings.TrimSpace(f)
		if f == "" || nonSku[f] {
			continue
		}
		features = append(features, f)
	}
	return features
}

func setFeaturesQuery() {
	paidFeatureSuffix = configOptions.GetString(config.Keys.PaidFeatureSuffix)

	// Query the base feature and its "_paid" variant for every SKU feature. Feature Service
	// returns an empty result for features that do not exist, so querying a "_paid" variant
	// that has not been defined is harmless.
	var features []string
	for _, feature := range skuFeatures() {
		features = append(features, feature, feature+paidFeatureSuffix)
	}

	featuresQuery = "?features=" + strings.Join(features, "&features=")
}

// InitPaidFeatures loads the paid-feature catalog once at startup. Doing this here rather
// than lazily in the request path avoids a data race on the paidFeatures package var under
// concurrent first requests, and removes first-request latency. Must be called after
// SetBundleInfo, since it depends on the SKU/non-SKU bundle split. It fails safe: on any
// error paidFeatures is an empty (non-nil) map, so is_trial degrades to false.
func InitPaidFeatures() {
	paidFeatures = loadPaidFeatures()
}

// loadPaidFeatures queries the Feature Service catalog (/features/v1) to determine which
// SKU features have a corresponding "_paid" variant. The result is used only to decide
// is_trial. It always returns a non-nil map; on any failure it returns an empty map so
// is_trial fails safe to false — entitlement correctness does not depend on this call.
func loadPaidFeatures() map[string]bool {
	result := make(map[string]bool)

	suffix := configOptions.GetString(config.Keys.PaidFeatureSuffix)
	url := fmt.Sprintf("%s%s",
		configOptions.GetString(config.Keys.SubsHost),
		configOptions.GetString(config.Keys.FeaturesAPIPath),
	)

	resp, err := getClient().Get(url)
	if err != nil {
		l.Log.WithFields(logrus.Fields{"error": err, "url": url}).Error("Unable to load feature catalog for trial detection")
		sentry.CaptureException(err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		l.Log.WithFields(logrus.Fields{"code": resp.StatusCode, "url": url}).Error("Non-200 loading feature catalog for trial detection")
		return result
	}

	body, _ := io.ReadAll(resp.Body)
	var catalog types.FeatureStatus
	if err := json.Unmarshal(body, &catalog); err != nil {
		l.Log.WithFields(logrus.Fields{"error": err}).Error("Unable to parse feature catalog for trial detection")
		sentry.CaptureException(err)
		return result
	}

	catalogNames := make(map[string]bool, len(catalog.Features))
	for _, f := range catalog.Features {
		catalogNames[f.Name] = true
	}

	for _, f := range skuFeatures() {
		result[f] = catalogNames[f+suffix]
	}

	return result
}

// GetFeatureStatus calls the IT feature service features endpoint and returns the entitlements for specified features/bundles
var GetFeatureStatus = func(params GetFeatureStatusParams) types.FeatureResponse {
	orgID := params.OrgId
	item := cache.Get(orgID)
	entitleAll := configOptions.GetString(config.Keys.EntitleAll)

	if item != nil && !item.Expired() && !params.ForceFreshData {
		return types.FeatureResponse{
			StatusCode: 200,
			Data:       item.Value(),
			CacheHit:   true,
		}
	}

	if entitleAll == "true" {
		return types.FeatureResponse{
			StatusCode: 200,
			Data:       types.FeatureStatus{},
			CacheHit:   false,
		}
	}

	if featuresQuery == "" { // build the static part of our query only once
		setFeaturesQuery()
	}

	req := fmt.Sprintf("%s%s%s&accountId=%s",
			configOptions.GetString(config.Keys.SubsHost),
			configOptions.GetString(config.Keys.FeatureStatusAPIPath),
			featuresQuery,
			orgID,
		)

	resp, err := getClient().Get(req)

	if err != nil {
		sentry.CaptureException(err)
		// cache fail-closed state to avoid repeated downstream calls until TTL expires
		cache.Set(orgID, types.FeatureStatus{}, cacheDuration)
		return types.FeatureResponse{
			StatusCode: 0,
			Error:      err,
			Data:       types.FeatureStatus{},
			CacheHit:   false,
			Url:        req,
		}
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		// cache fail-closed state to avoid repeated downstream calls until TTL expires
		cache.Set(orgID, types.FeatureStatus{}, cacheDuration)
		return types.FeatureResponse{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			Error:      nil,
			Data:       types.FeatureStatus{},
			CacheHit:   false,
			Url:        req,
		}
	}

	defer resp.Body.Close()

	// Unmarshaling response from Feature service
	body, _ := io.ReadAll(resp.Body)
	var FeatureStatus types.FeatureStatus
	json.Unmarshal(body, &FeatureStatus)

	cache.Set(orgID, FeatureStatus, cacheDuration)

	return types.FeatureResponse{
		StatusCode: resp.StatusCode,
		Data:       FeatureStatus,
		CacheHit:   false,
		Url:        req,
	}
}

func failOnDependencyError(errMsg string, res types.FeatureResponse, w http.ResponseWriter) {
	dependencyError := types.DependencyErrorDetails{
		DependencyFailure: true,
		Service:           "Feature Service",
		Status:            res.StatusCode,
		Endpoint:          configOptions.GetString(config.Keys.SubsHost),
		Message:           errMsg,
	}

	errorResponse := types.DependencyErrorResponse{Error: dependencyError}
	errorResponsejson, _ := json.Marshal(errorResponse)

	subsFailure.WithLabelValues(strconv.Itoa(res.StatusCode)).Inc()
	http.Error(w, string(errorResponsejson), 500)
}

func setBundlePayload(entitle bool, trial bool) types.EntitlementsSection {
	return types.EntitlementsSection{IsEntitled: entitle, IsTrial: trial}
}

// Represents a fail-closed state (empty feature set cached after failure).
func isCachedFailClosed(res types.FeatureResponse) bool {
	return res.CacheHit && len(res.Data.Features) == 0
}

// Services the handler for GETs to /api/entitlements/v1/services/
func Services() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		idObj := identity.GetIdentity(req.Context()).Identity
		orgId := idObj.Internal.OrgID

		queryParams := GetServicesParams{
			IncludeBundles: filtersFromParams(req, IncludeBundlesParamKey),
			ExcludeBundles: filtersFromParams(req, ExcludeBundlesParamKey),
			TrialActivated: boolFromParams(req, TrialActivatedParamKey),
		}
		subscriptions := GetFeatureStatus(
			GetFeatureStatusParams{
				OrgId:          orgId,
				ForceFreshData: queryParams.TrialActivated,
			},
		)

		subscriptionsMap := make(map[string]types.Feature)
		for _, feature := range subscriptions.Data.Features {
			subscriptionsMap[feature.Name] = feature
		}

		degraded := false
		if subscriptions.Error != nil {
			errMsg := "Unexpected error while talking to Feature Service"
			l.Log.WithFields(logrus.Fields{"error": subscriptions.Error}).Error(errMsg)
			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetTag("response_body", subscriptions.Body)
				scope.SetTag("response_status", strconv.Itoa(subscriptions.StatusCode))
				scope.SetTag("url", subscriptions.Url)
				sentry.CaptureException(fmt.Errorf("%s : %w", errMsg, subscriptions.Error))
			})
			// the request is degraded because we received an error from the feature service
			degraded = true
			subsFailure.WithLabelValues(strconv.Itoa(subscriptions.StatusCode)).Inc()
		}

		accNum := idObj.AccountNumber

		// For Service Accounts, User field is nil
		isInternal := false
		validEmailMatch := false
		if idObj.User != nil {
			isInternal = idObj.User.Internal
			validEmailMatch, _ = regexp.MatchString(`^.*@redhat.com$`, idObj.User.Email)
		}

		validAccNum := !(accNum == "" || accNum == "-1")
		validOrgId := !(orgId == "" || orgId == "-1")

		include_filter := queryParams.IncludeBundles
		exclude_filter := queryParams.ExcludeBundles

		subsTimeTaken := time.Since(start).Seconds()
		l.Log.WithFields(logrus.Fields{
			"subs_call_duration": subsTimeTaken,
			"cache_hit":          subscriptions.CacheHit,
			"url":                subscriptions.Url,
			"org_id":             orgId,
		}).Info("feature service call complete")
		subsTimeHistogram.Observe(subsTimeTaken)

		if subscriptions.Error == nil && subscriptions.StatusCode != 200 {
			errMsg := "Got back a non 200 status code from Feature Service"
			l.Log.WithFields(logrus.Fields{"code": subscriptions.StatusCode, "body": subscriptions.Body}).Error(errMsg)

			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetTag("response_body", subscriptions.Body)
				scope.SetTag("response_status", strconv.Itoa(subscriptions.StatusCode))
				scope.SetTag("url", subscriptions.Url)
				sentry.CaptureException(errors.New(errMsg))
			})

			// the request is degraded because we received a non-200 from the feature service
			degraded = true
			subsFailure.WithLabelValues(strconv.Itoa(subscriptions.StatusCode)).Inc()
		}

		if isCachedFailClosed(subscriptions) {
			degraded = true
		}

		entitleAll := configOptions.GetBool(config.Keys.EntitleAll)

		passesFilter := func(name string) bool {
			if len(include_filter) > 0 {
				return slices.Contains(include_filter, name)
			}
			if len(exclude_filter) > 0 {
				return !slices.Contains(exclude_filter, name)
			}
			return true
		}

		entitlementsResponse := make(map[string]types.EntitlementsSection)

		// SKU-based features are sourced from ENT_FEATURES and resolved against Feature
		// Service. is_trial is true only for paid-capable features where the base feature is
		// present but the "_paid" variant is not.
		for _, feature := range skuFeatures() {
			if !passesFilter(feature) {
				continue
			}
			if entitleAll {
				entitlementsResponse[feature] = setBundlePayload(true, false)
				continue
			}

			_, isEntitled := subscriptionsMap[feature]
			isTrial := false
			if isEntitled && paidFeatures[feature] {
				_, paidFeatExists := subscriptionsMap[feature+paidFeatureSuffix]
				isTrial = !paidFeatExists
			}
			entitlementsResponse[feature] = setBundlePayload(isEntitled, isTrial)
		}

		// Non-SKU bundles are sourced from bundles.yml. SKU-based bundles are handled above
		// via ENT_FEATURES (and are being removed from bundles.yml), so skip them here.
		for _, bundle := range bundleInfo {
			if bundle.IsSkuBased() {
				continue
			}
			if !passesFilter(bundle.Name) {
				continue
			}
			if entitleAll {
				entitlementsResponse[bundle.Name] = setBundlePayload(true, false)
				continue
			}

			isEntitled := true
			if bundle.UseValidAccNum {
				isEntitled = validAccNum && isEntitled
			}
			if bundle.UseValidOrgId {
				isEntitled = validOrgId && isEntitled
			}
			if bundle.UseIsInternal {
				isEntitled = validAccNum && isInternal && validEmailMatch
			}
			entitlementsResponse[bundle.Name] = setBundlePayload(isEntitled, false)
		}

		obj, err := json.Marshal(entitlementsResponse)

		if err != nil {
			l.Log.WithFields(logrus.Fields{"error": err}).Error("Unexpected error while unmarshalling JSON data from Subs Service")
			sentry.CaptureException(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if degraded {
			w.Header().Set("X-Entitlements-Degraded", "true")
			statusForHeader := subscriptions.StatusCode
			if isCachedFailClosed(subscriptions) {
				statusForHeader = 0
			}
			w.Header().Set("X-Entitlements-Degraded-Status", strconv.Itoa(statusForHeader))
		}
		w.Write([]byte(obj))
	}
}

func filtersFromParams(req *http.Request, filterName string) []string {
	var filter []string
	list := req.URL.Query().Get(filterName)
	if list != "" {
		filter = strings.Split(list, ",")
	}
	return filter
}

func boolFromParams(req *http.Request, paramName string) bool {
	strParam := req.URL.Query().Get(paramName)

	if strParam == "" {
		return false
	}

	param, err := strconv.ParseBool(strParam)

	if err != nil {
		return false
	}

	return param
}
