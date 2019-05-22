package controllers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/RedHatInsights/entitlements-api-go/controllers"
	. "github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/RedHatInsights/platform-go-middlewares/identity"
)

const DEFAULT_ORG_ID string = "4384938490324"

func testRequest(method string, path string, orgid string, fakeCaller func(string) SubscriptionsResponse) (*httptest.ResponseRecorder, EntitlementsResponse, string) {
	req, err := http.NewRequest(method, path, nil)
	Expect(err).To(BeNil(), "NewRequest error was not nil")

	ctx := context.Background()
	ctx = context.WithValue(ctx, identity.Key, identity.XRHID {
		Identity: identity.Identity {
			AccountNumber: "540155",
			Internal: identity.Internal {
				OrgID: orgid,
			},
		},
	})

	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	Index(fakeCaller)(rr, req)

	out, err := ioutil.ReadAll(rr.Result().Body)
	Expect(err).To(BeNil(), "ioutil.ReadAll error was not nil")

	rr.Result().Body.Close()

	var ret EntitlementsResponse
	json.Unmarshal(out, &ret)

	return rr, ret, string(out)
}

func testRequestWithDefaultOrgId(method string, path string, fakeCaller func(string) SubscriptionsResponse) (*httptest.ResponseRecorder, EntitlementsResponse, string) {
	return testRequest(method, path, DEFAULT_ORG_ID, fakeCaller)
}

func fakeGetSubscriptions(expetedOrgID string, response SubscriptionsResponse) func(string) SubscriptionsResponse {
	return func(orgID string) SubscriptionsResponse {
		Expect(expetedOrgID).To(Equal(orgID))
		return response
	}
}

func expectPass(res *http.Response) {
	Expect(res.StatusCode).To(Equal(200))
	Expect(res.Header.Get("Content-Type")).To(Equal("application/json"))
}

var _ = Describe("Identity Controller", func() {
	It("should call GetSubscriptions with the org_id on the context", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       []string{"foo", "bar"},
			CacheHit:   false,
		}
		testRequest("GET", "/", "540155", fakeGetSubscriptions("540155", fakeResponse))
		testRequest("GET", "/", "deadbeef12", fakeGetSubscriptions("deadbeef12", fakeResponse))
	})

	Context("When the Subs API sends back an error", func() {
		It("should fail the response", func() {
			rr, _, rawBody := testRequestWithDefaultOrgId("GET", "/", func(string) SubscriptionsResponse {
				return SubscriptionsResponse{StatusCode: 500, Data: nil, CacheHit: false}
			})

			Expect(rr.Result().StatusCode).To(Equal(500))
			Expect(rawBody).To(ContainSubstring(http.StatusText(500)))
		})
	})

	Context("When the Subs API says we have Smart Managment", func() {
		It("should give back a valid EntitlementsResponse with all is_entitled true", func() {
			fakeResponse := SubscriptionsResponse{
				StatusCode: 200,
				Data:       []string{"foo", "bar"},
				CacheHit:   false,
			}

			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartMangement.IsEntitled).To(Equal(true), "smart_management.is_entitled expected to be true")
		})
	})

	Context("When the Subs API says we *dont* have Smart Managment", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       []string{},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with smart_management false", func() {
			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartMangement.IsEntitled).To(Equal(false), "smart_management.is_entitled expected to be false")
		})
	})
})
