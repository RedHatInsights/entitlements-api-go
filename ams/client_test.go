package ams

import (
	"net/http"
	"net/url"

	"github.com/RedHatInsights/entitlements-api-go/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("AMS Client", func() {

	var amsServer 			*ghttp.Server
	var tokenServer			*ghttp.Server

	BeforeEach(func() {
		tokenServer = ghttp.NewServer()
		tokenServer.AppendHandlers(ghttp.RespondWith(http.StatusOK, `{
			"access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			"expires_in": 99999,
			"refresh_expires_in": 99999,
			"refresh_token": "refresh",
			"token_type": "Bearer",
			"not-before-policy": 0,
			"session_state": "state",
			"scope": ""
		}`, http.Header{"Content-Type": {"application/json"}}))			
		
		amsServer = ghttp.NewServer()
		amsServer.Writer = GinkgoWriter

		// this points our ams client to our mock servers setup above
		cfg := config.GetConfig().Options
		cfg.SetDefault(config.Keys.AMSHost, amsServer.URL())
		cfg.SetDefault(config.Keys.ClientID, "client-id")
		cfg.SetDefault(config.Keys.ClientSecret, "client-secret")
		cfg.SetDefault(config.Keys.TokenURL, tokenServer.URL())
	})

	AfterEach(func() {
		amsServer.Close()
		tokenServer.Close()
	})
	
	Context("GetSubscriptions", func() {

		BeforeEach(func() {
			amsServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/accounts_mgmt/v1/organizations"),
					ghttp.RespondWith(http.StatusOK, `{"items":[{"id": "amsOrgId"}]}`, http.Header{"Content-Type": {"application/json"}}),
				),
			)
		})

		It("should construct the base query correctly", func() {
			returnedSubs :=`{"items":[{"id": "subId", "status": "active"}]}`
			
			amsServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", ContainSubstring("/api/accounts_mgmt/v1/subscriptions")),
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						params, err := url.ParseQuery(r.URL.RawQuery)
						
						Expect(err).ToNot(HaveOccurred(), "query should be constructed with valid params")
						Expect(params).To(HaveLen(4))
						Expect(params.Has("search")).To(BeTrue(), "params should have search")
						Expect(params.Has("fetchAccounts")).To(BeTrue(), "params should have fetchAccounts")
						Expect(params.Has("size")).To(BeTrue(), "params should have size")
						Expect(params.Has("page")).To(BeTrue(), "params should have page")

						search := params.Get("search")
						Expect(search).To(Equal("plan.id LIKE 'AnsibleWisdom' AND organization_id = 'amsOrgId'"))
					}),
					ghttp.RespondWith(http.StatusOK, returnedSubs, http.Header{"Content-Type": {"application/json"}}),
				),
			)

			client, err := NewClient(false)
			Expect(err).To(BeNil())
			
			subs, err := client.GetSubscriptions("orgId", []string{}, 1, 0)

			Expect(err).To(BeNil())
			Expect(subs).ToNot(BeNil())
		})

		When("no statuses are included", func() {
			It("queries for subscriptions without filtering status", func() {
				returnedSubs :=`{"items":[{"id": "subId", "status": "active"}]}`
				
				amsServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/accounts_mgmt/v1/subscriptions"),
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							params, err := url.ParseQuery(r.URL.RawQuery)
							
							Expect(err).ToNot(HaveOccurred(), "query should be constructed with valid params")
							Expect(params.Has("search")).To(BeTrue(), "params should have search")
							
							search := params.Get("search")
							Expect(search).ToNot(ContainSubstring("status"))
						}),
						ghttp.RespondWith(http.StatusOK, returnedSubs, http.Header{"Content-Type": {"application/json"}}),
					),
				)

				client, err := NewClient(false)
				Expect(err).To(BeNil())
				
				subs, err := client.GetSubscriptions("orgId", []string{}, 1, 0)

				Expect(err).To(BeNil())
				Expect(subs).ToNot(BeNil())
			})
		})

		When("statuses are included", func() {
			It("queries for subscriptions with the desired status", func() {
				returnedSubs :=`{"items":[{"id": "subId", "status": "active"}]}`
				
				amsServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/accounts_mgmt/v1/subscriptions"),
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							params, err := url.ParseQuery(r.URL.RawQuery)
							
							Expect(err).ToNot(HaveOccurred(), "query should be constructed with valid params")
							Expect(params.Has("search")).To(BeTrue(), "params should have search")
							
							search := params.Get("search")
							Expect(search).To(Equal("plan.id LIKE 'AnsibleWisdom' AND organization_id = 'amsOrgId' AND status IN ('Active')"))
						}),
						ghttp.RespondWith(http.StatusOK, returnedSubs, http.Header{"Content-Type": {"application/json"}}),
					),
				)

				client, err := NewClient(false)
				Expect(err).To(BeNil())
				
				subs, err := client.GetSubscriptions("orgId", []string{"active"}, 1, 0)

				Expect(err).To(BeNil())
				Expect(subs).ToNot(BeNil())
			})
		})
	})
})