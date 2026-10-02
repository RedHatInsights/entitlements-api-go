package controllers

import (
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
	. "github.com/RedHatInsights/entitlements-api-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

// loadPaidFeatures derives, from the Feature Service catalog (/features/v1), which SKU
// features have a "_paid" variant. Only is_trial depends on it, so a failed load must
// fail safe to an empty (but non-nil) map, yielding is_trial=false rather than an error.
var _ = Describe("loadPaidFeatures", func() {
	var catalogServer *ghttp.Server
	var origSubsHost string
	var origFeatures string

	BeforeEach(func() {
		cfg := config.GetConfig().Options
		origSubsHost = cfg.GetString(config.Keys.SubsHost)
		origFeatures = cfg.GetString(config.Keys.Features)

		catalogServer = ghttp.NewServer()
		catalogServer.Writer = GinkgoWriter
		cfg.Set(config.Keys.SubsHost, catalogServer.URL())

		// Two SKU features under test: ansible (paid-capable) and rhods (skus-only).
		bundleInfo = []Bundle{}
		cfg.Set(config.Keys.Features, "ansible,rhods")
	})

	AfterEach(func() {
		catalogServer.Close()
		cfg := config.GetConfig().Options
		cfg.Set(config.Keys.SubsHost, origSubsHost)
		cfg.Set(config.Keys.Features, origFeatures)
	})

	It("derives paid-capability from the catalog: _paid present => true, absent => false", func() {
		catalogServer.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", config.GetConfig().Options.GetString(config.Keys.FeaturesAPIPath)),
			ghttp.RespondWith(http.StatusOK, `{"features": [
				{"name": "ansible"},
				{"name": "ansible_paid"},
				{"name": "rhods"}
			]}`, http.Header{"Content-Type": {"application/json"}}),
		))

		result := loadPaidFeatures()

		Expect(result).To(Equal(map[string]bool{
			"ansible": true,
			"rhods":   false,
		}))
		Expect(catalogServer.ReceivedRequests()).To(HaveLen(1))
	})

	It("fails safe to a non-nil empty map on a non-200 response", func() {
		catalogServer.AppendHandlers(ghttp.RespondWith(http.StatusServiceUnavailable, `down`))

		result := loadPaidFeatures()

		Expect(result).ToNot(BeNil())
		Expect(result).To(BeEmpty())
	})

	It("fails safe to a non-nil empty map on an unparseable body", func() {
		catalogServer.AppendHandlers(ghttp.RespondWith(http.StatusOK, `not json`,
			http.Header{"Content-Type": {"application/json"}}))

		result := loadPaidFeatures()

		Expect(result).ToNot(BeNil())
		Expect(result).To(BeEmpty())
	})
})
