package controllers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"errors"
	"testing"

	. "github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const DEFAULT_ORG_ID string = "4384938490324"
const DEFAULT_ACCOUNT_NUMBER string = "540155"
const DEFAULT_IS_INTERNAL bool = false
const DEFAULT_EMAIL = "test+qa@redhat.com"

func testRequest(method string, path string, accnum string, orgid string, isinternal bool, email string, fakeCaller func(string) SubscriptionsResponse) (*httptest.ResponseRecorder, map[string]EntitlementsSection, string) {
	req, err := http.NewRequest(method, path, nil)
	Expect(err).To(BeNil(), "NewRequest error was not nil")

	ctx := context.Background()
	ctx = context.WithValue(ctx, identity.Key, identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: accnum,
			User: identity.User{
				Internal: isinternal,
				Email: email,
			},
			Internal: identity.Internal{
				OrgID: orgid,
			},
		},
	})

	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	GetFeatureStatus = fakeCaller

	Index()(rr, req)

	out, err := ioutil.ReadAll(rr.Result().Body)
	Expect(err).To(BeNil(), "ioutil.ReadAll error was not nil")

	rr.Result().Body.Close()

	var ret map[string]EntitlementsSection
	json.Unmarshal(out, &ret)

	return rr, ret, string(out)
}

func testRequestWithDefaultOrgId(method string, path string, fakeCaller func(string) SubscriptionsResponse) (*httptest.ResponseRecorder, map[string]EntitlementsSection, string) {
	return testRequest(method, path, DEFAULT_ACCOUNT_NUMBER, DEFAULT_ORG_ID, DEFAULT_IS_INTERNAL, DEFAULT_EMAIL, fakeCaller)
}

func fakeGetFeatureStatus(expectedOrgID string, response SubscriptionsResponse) func(string) SubscriptionsResponse {
	return func(orgID string) SubscriptionsResponse {
		Expect(expectedOrgID).To(Equal(orgID))
		return response
	}
}

func expectPass(res *http.Response) {
	Expect(res.StatusCode).To(Equal(200))
	Expect(res.Header.Get("Content-Type")).To(Equal("application/json"))
}

var _ = Describe("Identity Controller", func() {

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

	Context("When the Subs API sends back a non-200", func() {
		It("should fail the response", func() {
			rr, _, rawBody := testRequestWithDefaultOrgId("GET", "/", func(string) SubscriptionsResponse {
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
			rr, _, rawBody := testRequestWithDefaultOrgId("GET", "/", func(string) SubscriptionsResponse {
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
				Data:       FeatureStatus{
					[]Feature{
						{ Name: "TestBundle1", IsEval: false, Entitled: false },
						{ Name: "TestBundle2", IsEval: true,  Entitled: true },
					},
				},
				CacheHit:   false,
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
