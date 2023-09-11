package ams

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/RedHatInsights/entitlements-api-go/api"
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

			params := api.GetSeatsParams{
				Status: &[]string{""},
			}
			
			subs, err := client.GetSubscriptions("orgId", params, 1, 0)

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

				params := api.GetSeatsParams{
					Status: &[]string{""},
				}
				
				subs, err := client.GetSubscriptions("orgId", params, 1, 0)

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

				params := api.GetSeatsParams{
					Status: &[]string{"active"},
				}
				
				subs, err := client.GetSubscriptions("orgId", params, 1, 0)

				Expect(err).To(BeNil())
				Expect(subs).ToNot(BeNil())
			})

			Context("and status is unsupported", func() {
				It("returns an error and does not query ams", func() {
					client, err := NewClient(false)
					Expect(err).To(BeNil())
					
					params := api.GetSeatsParams{
						Status: &[]string{"active", "inactive"},
					}

					subs, err := client.GetSubscriptions("orgId", params, 1, 0)
	
					Expect(subs).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("provided status 'inactive' is an unsupported status"))
					Expect(amsServer.ReceivedRequests()).To(HaveLen(1))

					var clientError *ClientError
					Expect(err).To(BeAssignableToTypeOf(clientError))
					errors.As(err, &clientError)
					Expect(clientError.StatusCode).To(BeEquivalentTo(http.StatusBadRequest))
				})
			})
		})
	})

	Context("ConvertUserOrgId", func() {
		var client AMSInterface

		BeforeEach(func() {
			var err error
			client, err = NewClient(false)
			Expect(err).ToNot(HaveOccurred())
		})

		When("cache is cold", func() {
			It("returns org id from service", func() {
				amsServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/accounts_mgmt/v1/organizations"),
						ghttp.RespondWith(http.StatusOK, `{"items":[{"id": "amsOrgId"}]}`, http.Header{"Content-Type": {"application/json"}}),
					),
				)

				amsOrgId, err :=client.ConvertUserOrgId("orgId")

				Expect(err).ToNot(HaveOccurred())
				Expect(amsOrgId).ToNot(BeNil())
				Expect(amsServer.ReceivedRequests()).To(HaveLen(1))
			})
		})

		When("cache is hot", func() {
			It("returns org id from cache", func() {
				amsServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/accounts_mgmt/v1/organizations"),
						ghttp.RespondWith(http.StatusOK, `{"items":[{"id": "amsOrgId"}]}`, http.Header{"Content-Type": {"application/json"}}),
					),
				)

				client.ConvertUserOrgId("orgId")
				amsOrgId, err := client.ConvertUserOrgId("orgId")

				Expect(err).ToNot(HaveOccurred())
				Expect(amsOrgId).ToNot(BeNil())
				Expect(amsServer.ReceivedRequests()).To(HaveLen(1))
			})
		})

		When("no ams org id found", func() {
			It("returns an error", func() {
				orgId := "orgId"

				amsServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/accounts_mgmt/v1/organizations"),
						ghttp.RespondWith(http.StatusOK, `{"items":[]}`, http.Header{"Content-Type": {"application/json"}}),
					),
				)

				amsOrgId, err := client.ConvertUserOrgId(orgId)

				Expect(amsOrgId).To(BeEmpty())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no corresponding ams org id found"))
				Expect(amsServer.ReceivedRequests()).To(HaveLen(1))
				
				var clientError *ClientError
				Expect(err).To(BeAssignableToTypeOf(clientError))
				errors.As(err, &clientError)
				Expect(clientError.StatusCode).To(BeEquivalentTo(http.StatusBadRequest))
				Expect(clientError.OrgId).To(BeEquivalentTo(orgId))
				Expect(clientError.AmsOrgId).To(BeEmpty())
			})
		})

		When("invalid user org id used", func() {
			It("returns an error", func() {
				orgId 		:= "org-id"

				amsOrgId, err := client.ConvertUserOrgId(orgId)

				Expect(amsOrgId).To(BeEmpty())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid user org id"))
				Expect(amsServer.ReceivedRequests()).To(HaveLen(0))
				
				var clientError *ClientError
				Expect(err).To(BeAssignableToTypeOf(clientError))
				errors.As(err, &clientError)
				Expect(clientError.StatusCode).To(BeEquivalentTo(http.StatusInternalServerError))
				Expect(clientError.OrgId).To(BeEquivalentTo(orgId))
				Expect(clientError.AmsOrgId).To(BeEmpty())
			})
		})

		When("invalid ams org id returned", func() {
			It("returns an error", func() {
				orgId 		:= "orgId"
				amsOrgId 	:= "ams-org-id"

				amsServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/accounts_mgmt/v1/organizations"),
						ghttp.RespondWith(http.StatusOK, `{"items":[{"id": "`+ amsOrgId +`"}]}`, http.Header{"Content-Type": {"application/json"}}),
					),
				)

				amsOrgId, err := client.ConvertUserOrgId(orgId)

				Expect(amsOrgId).To(BeEmpty())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid ams org id"))
				Expect(amsServer.ReceivedRequests()).To(HaveLen(1))
				
				var clientError *ClientError
				Expect(err).To(BeAssignableToTypeOf(clientError))
				errors.As(err, &clientError)
				Expect(clientError.StatusCode).To(BeEquivalentTo(http.StatusInternalServerError))
				Expect(clientError.OrgId).To(BeEquivalentTo(orgId))
				Expect(clientError.AmsOrgId).To(BeEquivalentTo("ams-org-id"))
			})
		})
	})
})