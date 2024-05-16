package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/RedHatInsights/entitlements-api-go/config"
	. "github.com/RedHatInsights/entitlements-api-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

const DEFAULT_ORG_ID string = "4384938490324"
const DEFAULT_ACCOUNT_NUMBER string = "540155"
const DEFAULT_IS_INTERNAL bool = false
const DEFAULT_EMAIL = "test+qa@redhat.com"

var realGetFeatureStatus = GetFeatureStatus

func testRequest(method string, path string, accnum string, orgid string, isinternal bool, email string, fakeCaller func(GetFeatureStatusParams) SubscriptionsResponse) (*httptest.ResponseRecorder, map[string]EntitlementsSection, string) {
	req, err := http.NewRequest(method, path, nil)
	Expect(err).To(BeNil(), "NewRequest error was not nil")

	ctx := context.Background()
	ctx = context.WithValue(ctx, identity.Key, identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: accnum,
			User: identity.User{
				Internal: isinternal,
				Email:    email,
			},
			Internal: identity.Internal{
				OrgID: orgid,
			},
		},
	})

	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	GetFeatureStatus = fakeCaller

	Services()(rr, req)

	out, err := io.ReadAll(rr.Result().Body)
	Expect(err).To(BeNil(), "io.ReadAll error was not nil")

	rr.Result().Body.Close()

	var ret map[string]EntitlementsSection
	json.Unmarshal(out, &ret)

	return rr, ret, string(out)
}

func testRequestWithDefaultOrgId(method string, path string, fakeCaller func(GetFeatureStatusParams) SubscriptionsResponse) (*httptest.ResponseRecorder, map[string]EntitlementsSection, string) {
	return testRequest(method, path, DEFAULT_ACCOUNT_NUMBER, DEFAULT_ORG_ID, DEFAULT_IS_INTERNAL, DEFAULT_EMAIL, fakeCaller)
}

func fakeGetFeatureStatus(expectedOrgID string, response SubscriptionsResponse) func(GetFeatureStatusParams) SubscriptionsResponse {
	return func(params GetFeatureStatusParams) SubscriptionsResponse {
		Expect(expectedOrgID).To(Equal(params.OrgId))
		return response
	}
}

func expectPass(res *http.Response) {
	Expect(res.StatusCode).To(Equal(200))
	Expect(res.Header.Get("Content-Type")).To(Equal("application/json"))
}

var _ = Describe("Services Controller", func() {

	BeforeEach(func() {
		bundleInfo = []Bundle{}
		if err := SetBundleInfo("../test_data/test_bundle.yml"); err != nil {
			panic("Error in test_bundle.yml")
		}
	})

	It("should call GetFeatureStatus with the org_id on the context", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       FeatureStatus{},
			CacheHit:   false,
		}
		testRequest("GET", "/", DEFAULT_ACCOUNT_NUMBER, "540155", DEFAULT_IS_INTERNAL, DEFAULT_EMAIL, fakeGetFeatureStatus("540155", fakeResponse))
		testRequest("GET", "/", DEFAULT_ACCOUNT_NUMBER, "deadbeef12", DEFAULT_IS_INTERNAL, DEFAULT_EMAIL, fakeGetFeatureStatus("deadbeef12", fakeResponse))
	})

	It("should build the subscriptions query with only sku based features", func() {
		cfg := config.GetConfig()
		cfg.Options.Set(config.Keys.Features, "TestBundle1,TestBundle3,TestBundle4,TestBundle5,TestBundle6,TestBundle7")
		setSubscriptionsQueryFeatures()
		Expect(subsQueryFeatures).To(BeEquivalentTo("?features=TestBundle1&features=TestBundle6"))
	})

	Context("When the Subs API sends back a non-200", func() {
		It("should fail the response", func() {
			rr, _, rawBody := testRequestWithDefaultOrgId("GET", "/", func(GetFeatureStatusParams) SubscriptionsResponse {
				return SubscriptionsResponse{StatusCode: 503, Data: FeatureStatus{}, CacheHit: false}
			})

			var jsonResponse DependencyErrorResponse
			json.Unmarshal([]byte(rawBody), &jsonResponse)

			Expect(rr.Result().StatusCode).To(Equal(500))
			Expect(jsonResponse.Error.DependencyFailure).To(Equal(true))
			Expect(jsonResponse.Error.Service).To(Equal("Subscriptions Service"))
			Expect(jsonResponse.Error.Status).To(Equal(503))
			Expect(jsonResponse.Error.Endpoint).To(Equal("https://subscription.api.redhat.com"))
			Expect(jsonResponse.Error.Message).To(Equal("Got back a non 200 status code from Subscriptions Service"))
		})
	})

	Context("When the Subs API sends back an error", func() {
		It("should fail the response", func() {
			rr, _, rawBody := testRequestWithDefaultOrgId("GET", "/", func(GetFeatureStatusParams) SubscriptionsResponse {
				return SubscriptionsResponse{StatusCode: 503, Data: FeatureStatus{}, CacheHit: false, Error: errors.New("Sub Failure")}
			})

			var jsonResponse DependencyErrorResponse
			json.Unmarshal([]byte(rawBody), &jsonResponse)

			Expect(rr.Result().StatusCode).To(Equal(500))
			Expect(jsonResponse.Error.DependencyFailure).To(Equal(true))
			Expect(jsonResponse.Error.Service).To(Equal("Subscriptions Service"))
			Expect(jsonResponse.Error.Status).To(Equal(503))
			Expect(jsonResponse.Error.Endpoint).To(Equal("https://subscription.api.redhat.com"))
			Expect(jsonResponse.Error.Message).To(Equal("Unexpected error while talking to Subs Service"))
		})
	})

	Context("When the bundles.yml has errors", func() {
		It("should include errors when file is not available", func() {
			bundleInfo = []Bundle{}
			err := SetBundleInfo("no_such_file")
			Expect(len(bundleInfo)).To(Equal(0))
			Expect(err).ToNot(Equal(nil))
		})

		It("should return error for yaml parse errors", func() {
			bundleInfo = []Bundle{}
			err := SetBundleInfo("../test_data/err_bundle.yml")
			Expect(len(bundleInfo)).To(Equal(0))
			Expect(err).ToNot(Equal(nil))
		})
	})

	Context("When the account number is -1 or '' ", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       FeatureStatus{},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with bundles using Valid Account Number false", func() {
			// testing with account number "-1"
			rr, body, _ := testRequest("GET", "/", "-1", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle3"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle4"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle5"].IsEntitled).To(Equal(false))
		})

		It("should give back a valid EntitlementsResponse with bundles using Valid Account Number false", func() {
			// testing with account number ""
			rr, body, _ := testRequest("GET", "/", "", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle3"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle4"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle5"].IsEntitled).To(Equal(false))
		})
	})

	Context("When a bundle uses only Valid Account Number", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       FeatureStatus{},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with that bundle true", func() {
			// testing with account number ""
			rr, body, _ := testRequest("GET", "/", "123456", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle3"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle4"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle5"].IsEntitled).To(Equal(true))
		})

	})

	Context("When a bundle is defined with use_valid_org_id", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       FeatureStatus{},
			CacheHit:   false,
		}

		Context("When the org_id is invalid", func() {
			It("should not entitle bundles when a -1 org_id is supplied", func() {
				rr, body, _ := testRequest("GET", "/", DEFAULT_ACCOUNT_NUMBER, "-1", true, DEFAULT_EMAIL, fakeGetFeatureStatus("-1", fakeResponse))
				expectPass(rr.Result())
				Expect(body["TestBundle7"].IsEntitled).To(Equal(false))
			})

			It("should not entitle bundles when a blank org_id is supplied", func() {
				rr, body, _ := testRequest("GET", "/", DEFAULT_ACCOUNT_NUMBER, "", true, DEFAULT_EMAIL, fakeGetFeatureStatus("", fakeResponse))
				expectPass(rr.Result())
				Expect(body["TestBundle7"].IsEntitled).To(Equal(false))
			})
		})

		Context("When the org_id is valid", func() {
			It("should entitle bundles when a valid org_id is supplied", func() {
				rr, body, _ := testRequest("GET", "/", "123456", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
				expectPass(rr.Result())
				Expect(body["TestBundle7"].IsEntitled).To(Equal(true))
			})
		})
	})

	Context("When a bundle uses only use_is_internal", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       FeatureStatus{},
			CacheHit:   false,
		}

		It("should entitle when valid account and principal is internal", func() {
			rr, body, _ := testRequest("GET", "/", "123456", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle5"].IsEntitled).To(Equal(true))
		})

		It("should not entitle when valid account and principal is internal but email is not @redhat.com", func() {
			rr, body, _ := testRequest("GET", "/", "123456", DEFAULT_ORG_ID, true, "jdoe@example.com", fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle5"].IsEntitled).To(Equal(false))
		})

		It("should not entitle when valid account and principal is not internal", func() {
			rr, body, _ := testRequest("GET", "/", "123456", DEFAULT_ORG_ID, false, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle5"].IsEntitled).To(Equal(false))
		})

		It("should not entitle when not a valid account and principal is internal", func() {
			rr, body, _ := testRequest("GET", "/", "", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle5"].IsEntitled).To(Equal(false))
		})

		It("should not entitle when not a valid account and principal is internal", func() {
			rr, body, _ := testRequest("GET", "/", "-1", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle5"].IsEntitled).To(Equal(false))
		})
	})

	Context("When the Subscriptions API returns features", func() {
		It("should set values based on response from the featureStatus request", func() {
			fakeResponse := SubscriptionsResponse{
				StatusCode: 200,
				Body:       "",
				Error:      nil,
				Data: FeatureStatus{
					Features: []Feature{
						{
							Name:     "TestBundle1",
							IsEval:   false,
							Entitled: false,
						},
						{
							Name:     "TestBundle2",
							IsEval:   true,
							Entitled: true,
						},
					},
				},
				CacheHit: false,
			}

			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsTrial).To(Equal(false))
			Expect(body["TestBundle2"].IsTrial).To(Equal(true))
			Expect(body["TestBundle6"].IsTrial).To(Equal(false))
			Expect(body["TestBundle1"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle6"].IsEntitled).To(Equal(false))
		})
	})

	Context("When the request contains query filters", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       FeatureStatus{},
			CacheHit:   false,
		}
		It("should only return bundles included in include_bundles", func() {
			rr, body, _ := testRequest("GET", "/?include_bundles=TestBundle2,TestBundle3", "123456", DEFAULT_ORG_ID, false, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(len(body)).To(Equal(2))
			_, found := body["TestBundle1"]
			Expect(found).To((BeFalse()))
			_, found = body["TestBundle2"]
			Expect(found).To(BeTrue())
			_, found = body["TestBundle3"]
			Expect(found).To(BeTrue())
		})
		It("should not return bundles included in exclude_bundles", func() {
			rr, body, _ := testRequest("GET", "/?exclude_bundles=TestBundle2,TestBundle3", "123456", DEFAULT_ORG_ID, false, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(len(body)).To(Equal(5))
			_, found := body["TestBundle1"]
			Expect(found).To((BeTrue()))
			_, found = body["TestBundle2"]
			Expect(found).To(BeFalse())
			_, found = body["TestBundle3"]
			Expect(found).To(BeFalse())
		})
		It("should handle single include_filter entries", func() {
			rr, body, _ := testRequest("GET", "/?include_bundles=TestBundle2", "123456", DEFAULT_ORG_ID, false, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(len(body)).To(Equal(1))
			_, found := body["TestBundle1"]
			Expect(found).To((BeFalse()))
			_, found = body["TestBundle2"]
			Expect(found).To(BeTrue())
			_, found = body["TestBundle3"]
			Expect(found).To(BeFalse())
		})
		It("Should handle single exclude_filter entries", func() {
			rr, body, _ := testRequest("GET", "/?exclude_bundles=TestBundle2", "123456", DEFAULT_ORG_ID, false, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(len(body)).To(Equal(6))
			_, found := body["TestBundle1"]
			Expect(found).To((BeTrue()))
			_, found = body["TestBundle2"]
			Expect(found).To(BeFalse())
			_, found = body["TestBundle3"]
			Expect(found).To(BeTrue())
		})
		It("should prioritize include_bundles", func() {
			rr, body, _ := testRequest("GET", "/?include_bundles=TestBundle1,TestBundle2&exclude_bundles=TestBundle2,TestBundle3", "123456", DEFAULT_ORG_ID, false, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(len(body)).To(Equal(2))
			_, found := body["TestBundle1"]
			Expect(found).To((BeTrue()))
			_, found = body["TestBundle2"]
			Expect(found).To(BeTrue())
			_, found = body["TestBundle3"]
			Expect(found).To(BeFalse())
		})
	})

	Context("When ENT_ENTITLE_ALL is set", func() {
		AfterEach(func() {
			os.Setenv("ENT_ENTITLE_ALL", "")
		})
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       FeatureStatus{},
			CacheHit:   false,
		}
		It("should skip IT calls and entitle all bundles when true", func() {
			os.Setenv("ENT_ENTITLE_ALL", "true")
			rr, body, _ := testRequest("GET", "/", "-1", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle3"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle4"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle5"].IsEntitled).To(Equal(true))
		})

		It("should return as normal when false", func() {
			os.Setenv("ENT_ENTITLE_ALL", "false")
			rr, body, _ := testRequest("GET", "/", "-1", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle3"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle4"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle5"].IsEntitled).To(Equal(false))
		})

		It("should return as normal when unset", func() {
			rr, body, _ := testRequest("GET", "/", "-1", DEFAULT_ORG_ID, true, DEFAULT_EMAIL, fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle3"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle4"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle5"].IsEntitled).To(Equal(false))
		})
	})

	Context("The request contains trial_activated", func() {
		When("trial_activated is correctly parsed from the query", func() {
			dummyResponse := SubscriptionsResponse{
				StatusCode: 200,
				Data:       FeatureStatus{},
				CacheHit:   false,
			}

			It("defaults the param to false when its absent", func() {
				// given
				var actualForceFreshData *bool
				mockGetFeatureStatus := func(params GetFeatureStatusParams) SubscriptionsResponse {
					actualForceFreshData = &params.ForceFreshData

					return dummyResponse
				}
				path := "/"

				// when
				testRequestWithDefaultOrgId("GET", path, mockGetFeatureStatus)

				// then
				Expect(actualForceFreshData).ToNot(BeNil())
				Expect(*actualForceFreshData).To(BeFalse())
			})

			It("defaults the param to false when its not a valid bool", func() {
				// given
				var actualForceFreshData *bool
				mockGetFeatureStatus := func(params GetFeatureStatusParams) SubscriptionsResponse {
					actualForceFreshData = &params.ForceFreshData

					return dummyResponse
				}
				path := "/?trial_activated=notABool"

				// when
				testRequestWithDefaultOrgId("GET", path, mockGetFeatureStatus)

				// then
				Expect(actualForceFreshData).ToNot(BeNil())
				Expect(*actualForceFreshData).To(BeFalse())
			})

			It("set the param to true when its a valid bool", func() {
				// given
				var actualForceFreshData *bool
				mockGetFeatureStatus := func(params GetFeatureStatusParams) SubscriptionsResponse {
					actualForceFreshData = &params.ForceFreshData

					return dummyResponse
				}
				path := "/?trial_activated=true"

				// when
				testRequestWithDefaultOrgId("GET", path, mockGetFeatureStatus)

				// then
				Expect(actualForceFreshData).ToNot(BeNil())
				Expect(*actualForceFreshData).To(BeTrue())
			})
		})

		When("trial_activated param is valid", func() {
			var subsServer *ghttp.Server

			BeforeEach(func() {
				GetFeatureStatus = realGetFeatureStatus

				// setup mock server
				subsServer = ghttp.NewServer()
				subsServer.Writer = GinkgoWriter

				// this will setup our mock server to respond with the following to the first request made to it,
				// aka the following request to fill cache
				subsServer.AppendHandlers(ghttp.RespondWith(http.StatusOK, `{"features": [
					{
						"name":"dummy feature",
						"isEval":false,
						"entitled":true
					}
				]}`, http.Header{"Content-Type": {"application/json"}}))

				// this points our http client to our mock server setup above
				cfg := config.GetConfig().Options
				cfg.SetDefault(config.Keys.SubsHost, subsServer.URL())

				// fill cache
				params := GetFeatureStatusParams{
					OrgId:          DEFAULT_ORG_ID,
					ForceFreshData: true,
				}
				GetFeatureStatus(params)
			})

			AfterEach(func() {
				subsServer.Close()
			})

			It("serves cached data when req param is false", func() {
				// given
				params := GetFeatureStatusParams{
					OrgId:          DEFAULT_ORG_ID,
					ForceFreshData: false,
				}

				// when
				response := GetFeatureStatus(params)

				// then
				Expect(response).ToNot(BeNil())
				Expect(response.CacheHit).To(BeTrue())
				Expect(subsServer.ReceivedRequests()).To(HaveLen(1))
			})

			It("serves fresh data when req param is true", func() {
				// given
				params := GetFeatureStatusParams{
					OrgId:          DEFAULT_ORG_ID,
					ForceFreshData: true,
				}

				subsServer.AppendHandlers(ghttp.RespondWith(http.StatusOK, `{"features": [
					{
						"name":"dummy feature 2!",
						"isEval":false,
						"entitled":true
					}
				]}`, http.Header{"Content-Type": {"application/json"}}))

				// when
				response := GetFeatureStatus(params)

				// then
				Expect(response).ToNot(BeNil())
				Expect(response.CacheHit).To(BeFalse())
				Expect(response.Data.Features).ToNot(BeNil())
				Expect(response.Data.Features).To(HaveLen(1))
				Expect(response.Data.Features[0].Name).To(BeEquivalentTo("dummy feature 2!"))
				Expect(subsServer.ReceivedRequests()).To(HaveLen(2))
			})
		})
	})
})

func BenchmarkRequest(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       FeatureStatus{},
			CacheHit:   false,
		}

		testRequestWithDefaultOrgId("GET", "/", fakeGetFeatureStatus(DEFAULT_ORG_ID, fakeResponse))
	}
}
