package controllers

import (
	"github.com/RedHatInsights/entitlements-api-go/config"
	. "github.com/RedHatInsights/entitlements-api-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// featureServiceFeatures resolves the set of bundles whose entitlement comes from Feature
// Service. The end state sources this purely from `use_feature_service: true` in
// bundles.yml; during the migration it is unioned with the legacy ENT_FEATURES env var.
var _ = Describe("featureServiceFeatures", func() {
	var origFeatures string
	var origBundleInfo []Bundle

	BeforeEach(func() {
		origFeatures = config.GetConfig().Options.GetString(config.Keys.Features)
		origBundleInfo = bundleInfo
	})

	AfterEach(func() {
		config.GetConfig().Options.Set(config.Keys.Features, origFeatures)
		bundleInfo = origBundleInfo
		featuresQuery = ""
	})

	It("resolves features from use_feature_service with ENT_FEATURES empty (end state)", func() {
		bundleInfo = []Bundle{
			{Name: "ansible", UseFeatureService: true},
			{Name: "insights", UseValidOrgId: true},
			{Name: "openshift", UseValidOrgId: false},
		}
		config.GetConfig().Options.Set(config.Keys.Features, "")

		Expect(featureServiceFeatures()).To(Equal([]string{"ansible"}))

		paidFeatureSuffix = config.GetConfig().Options.GetString(config.Keys.PaidFeatureSuffix)
		setFeaturesQuery()
		Expect(featuresQuery).To(Equal("?features=ansible&features=ansible_paid"))
	})

	It("unions the flag with legacy ENT_FEATURES and excludes non-flagged bundles (transition)", func() {
		bundleInfo = []Bundle{
			{Name: "ansible", UseFeatureService: true},
			{Name: "openshift", UseValidOrgId: false},
		}
		// ansible is de-duplicated (flag + ENT_FEATURES); rhods is added from ENT_FEATURES;
		// openshift is a non-flagged identity-gated bundle and must be excluded.
		config.GetConfig().Options.Set(config.Keys.Features, "ansible,rhods,openshift")

		Expect(featureServiceFeatures()).To(Equal([]string{"ansible", "rhods"}))
	})
})
