package controllers

import (
	"github.com/RedHatInsights/entitlements-api-go/config"
	. "github.com/RedHatInsights/entitlements-api-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// This suite locks the full /services response output against a prod-shaped bundle
// fixture so the SKU-deprecation refactor (RHCLOUD-49553) provably preserves behavior.
// It asserts the entire marshaled map[string]EntitlementsSection, not individual keys.
//
// The fixture's SKU bundle names match the prod ENT_FEATURES list (minus openshift),
// which is exactly the condition under which the old (bundleInfo-driven) and new
// (ENT_FEATURES-driven) response construction produce identical output.
var _ = Describe("Services response regression", func() {
	// prod-like ENT_FEATURES, deliberately including openshift to prove it is treated
	// as a non-SKU bundle (sourced from bundles.yml) and never queried as a SKU feature.
	const prodFeatures = "ansible,smart_management,rhods,rhoam,rhosak,openshift,acs"

	var origFeatures string

	BeforeEach(func() {
		bundleInfo = []Bundle{}
		if err := SetBundleInfo("../test_data/regression_bundle.yml"); err != nil {
			panic("Error loading regression_bundle.yml")
		}
		origFeatures = config.GetConfig().Options.GetString(config.Keys.Features)
		config.GetConfig().Options.Set(config.Keys.Features, prodFeatures)
		// paidFeatureSuffix is normally set as a side effect of setFeaturesQuery(), which
		// only runs inside the real GetFeatureStatus. GetFeatureStatus is stubbed here, so
		// pin the suffix explicitly to make trial detection deterministic regardless of
		// spec order.
		paidFeatureSuffix = config.GetConfig().Options.GetString(config.Keys.PaidFeatureSuffix)
		// paid-capable SKU features, as derived from the /features/v1 catalog in prod.
		// Pinning it avoids a live catalog call and matches the fixture's paid bundles.
		paidFeatures = map[string]bool{"ansible": true, "smart_management": true, "acs": true}
	})

	AfterEach(func() {
		config.GetConfig().Options.Set(config.Keys.Features, origFeatures)
	})

	// The non-SKU bundles in the fixture, all entitled for a fully-valid internal
	// Red Hat identity (valid acc num + valid org id + internal + @redhat.com email).
	nonSkuEntitled := func() map[string]EntitlementsSection {
		out := map[string]EntitlementsSection{}
		for _, name := range []string{
			"cost_management", "insights", "migrations", "subscriptions",
			"settings", "user_preferences", "openshift", "internal", "rhel",
		} {
			out[name] = EntitlementsSection{IsEntitled: true, IsTrial: false}
		}
		return out
	}

	sku := func(entitled, trial bool) EntitlementsSection {
		return EntitlementsSection{IsEntitled: entitled, IsTrial: trial}
	}

	// merge builds a full expected response map from the non-SKU baseline plus the
	// six SKU bundle expectations.
	merge := func(skus map[string]EntitlementsSection) map[string]EntitlementsSection {
		out := nonSkuEntitled()
		for k, v := range skus {
			out[k] = v
		}
		return out
	}

	featureResp := func(names ...string) FeatureResponse {
		features := make([]Feature, 0, len(names))
		for _, n := range names {
			features = append(features, Feature{Name: n})
		}
		return FeatureResponse{
			StatusCode: 200,
			Data:       FeatureStatus{Features: features},
			CacheHit:   false,
		}
	}

	// requestAsInternalRH issues the request as a fully-valid internal Red Hat user so
	// every non-SKU gate (acc num / org id / internal) evaluates to entitled.
	requestAsInternalRH := func(resp FeatureResponse) map[string]EntitlementsSection {
		_, body, _ := testRequest("GET", "/", "123456", DEFAULT_ORG_ID, true, DEFAULT_EMAIL,
			fakeGetFeatureStatus(DEFAULT_ORG_ID, resp))
		return body
	}

	It("returns all SKU bundles false when the user is entitled to nothing", func() {
		body := requestAsInternalRH(featureResp())
		expected := merge(map[string]EntitlementsSection{
			"ansible":          sku(false, false),
			"smart_management": sku(false, false),
			"acs":              sku(false, false),
			"rhods":            sku(false, false),
			"rhoam":            sku(false, false),
			"rhosak":           sku(false, false),
		})
		Expect(body).To(Equal(expected))
	})

	It("marks a paid-capable bundle entitled and not-trial when the _paid variant is present", func() {
		body := requestAsInternalRH(featureResp("ansible", "ansible_paid"))
		expected := merge(map[string]EntitlementsSection{
			"ansible":          sku(true, false),
			"smart_management": sku(false, false),
			"acs":              sku(false, false),
			"rhods":            sku(false, false),
			"rhoam":            sku(false, false),
			"rhosak":           sku(false, false),
		})
		Expect(body).To(Equal(expected))
	})

	It("marks a paid-capable bundle entitled and trial when only the base variant is present", func() {
		body := requestAsInternalRH(featureResp("ansible"))
		expected := merge(map[string]EntitlementsSection{
			"ansible":          sku(true, true),
			"smart_management": sku(false, false),
			"acs":              sku(false, false),
			"rhods":            sku(false, false),
			"rhoam":            sku(false, false),
			"rhosak":           sku(false, false),
		})
		Expect(body).To(Equal(expected))
	})

	It("marks a skus-only bundle entitled and NOT trial (rhods/rhoam/rhosak have no paid variant)", func() {
		body := requestAsInternalRH(featureResp("rhods", "rhoam", "rhosak"))
		expected := merge(map[string]EntitlementsSection{
			"ansible":          sku(false, false),
			"smart_management": sku(false, false),
			"acs":              sku(false, false),
			"rhods":            sku(true, false),
			"rhoam":            sku(true, false),
			"rhosak":           sku(true, false),
		})
		Expect(body).To(Equal(expected))
	})

	It("distinguishes acs trial (eval only) from acs paid", func() {
		trialBody := requestAsInternalRH(featureResp("acs"))
		Expect(trialBody["acs"]).To(Equal(sku(true, true)))

		paidBody := requestAsInternalRH(featureResp("acs", "acs_paid"))
		Expect(paidBody["acs"]).To(Equal(sku(true, false)))
	})

	It("fails SKU bundles closed and marks the response degraded on a non-200", func() {
		rr, body, _ := testRequestWithDefaultOrgId("GET", "/", func(GetFeatureStatusParams) FeatureResponse {
			return FeatureResponse{StatusCode: 503, Data: FeatureStatus{}, CacheHit: false}
		})
		Expect(rr.Result().Header.Get("X-Entitlements-Degraded")).To(Equal("true"))
		for _, name := range []string{"ansible", "smart_management", "acs", "rhods", "rhoam", "rhosak"} {
			Expect(body[name]).To(Equal(sku(false, false)), name+" should fail closed")
		}
	})

	It("openshift is entitled from bundles.yml and never treated as a SKU feature", func() {
		// openshift is in ENT_FEATURES but is a non-SKU bundle with use_valid_org_id: false,
		// so it must be entitled for everyone regardless of the feature service response.
		body := requestAsInternalRH(featureResp())
		Expect(body["openshift"]).To(Equal(EntitlementsSection{IsEntitled: true, IsTrial: false}))
	})

	It("honors include_bundles", func() {
		_, body, _ := testRequest("GET", "/?include_bundles=ansible,insights", "123456", DEFAULT_ORG_ID, true, DEFAULT_EMAIL,
			fakeGetFeatureStatus(DEFAULT_ORG_ID, featureResp("ansible", "ansible_paid")))
		Expect(body).To(Equal(map[string]EntitlementsSection{
			"ansible": sku(true, false),
			"insights": {IsEntitled: true, IsTrial: false},
		}))
	})

	It("honors exclude_bundles", func() {
		_, body, _ := testRequest("GET", "/?exclude_bundles=ansible,smart_management,acs,rhoam,rhosak", "123456", DEFAULT_ORG_ID, true, DEFAULT_EMAIL,
			fakeGetFeatureStatus(DEFAULT_ORG_ID, featureResp("rhods")))
		expected := merge(map[string]EntitlementsSection{"rhods": sku(true, false)})
		delete(expected, "ansible")
		delete(expected, "smart_management")
		delete(expected, "acs")
		delete(expected, "rhoam")
		delete(expected, "rhosak")
		Expect(body).To(Equal(expected))
	})
})
