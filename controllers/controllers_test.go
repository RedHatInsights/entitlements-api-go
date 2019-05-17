package controllers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"net/http"
	"net/http/httptest"
	"encoding/json"
	"io/ioutil"

	. "github.com/RedHatInsights/entitlements-api-go/controllers"
	"github.com/RedHatInsights/entitlements-api-go/types"
)

const DEFAULT_ORG_ID string = "4384938490324"

func testRequest(method string, path string, orgid string, fakeCaller func(string) []string) (*httptest.ResponseRecorder, types.EntitlementsResponse) {
	req, err := http.NewRequest(method, path, nil)
	Expect(err).To(BeNil(), "NewRequest error was not nil")

	ctx := context.Background()
	ctx = context.WithValue(ctx, "org_id", orgid)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	Index(fakeCaller)(rr, req)

	out, err := ioutil.ReadAll(rr.Result().Body)
	Expect(err).To(BeNil(), "ioutil.ReadAll error was not nil")

	rr.Result().Body.Close()
	var ret types.EntitlementsResponse
	json.Unmarshal(out, &ret)

	return rr, ret
}

func testRequestWithDefaultOrgId(method string, path string, fakeCaller func(string) []string) (*httptest.ResponseRecorder, types.EntitlementsResponse) {
	return testRequest(method, path, DEFAULT_ORG_ID, fakeCaller)
}

func fakeGetSubscriptions(expetedOrgID string, returnData []string) func(string) []string {
	return func(orgID string) []string {
		Expect(expetedOrgID).To(Equal(orgID))
		return returnData
	}
}

func expectPass(res *http.Response) {
	Expect(res.StatusCode).To(Equal(200))
	Expect(res.Header.Get("Content-Type")).To(Equal("application/json"))
}

var _ = Describe("Identity Controller", func() {
	It("should call GetSubscriptions with the org_id on the context", func() {
		testRequest("GET", "/", "540155",     fakeGetSubscriptions("540155", []string{"foo", "bar"}))
		testRequest("GET", "/", "deadbeef12", fakeGetSubscriptions("deadbeef12", []string{"foo", "bar"}))
	})

	Context("When the Subs API says we have Smart Managment", func() {
		It("should give back a valid EntitlementsResponse with all is_entitled true", func() {
			rr, body := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, []string{"foo", "bar"}))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartMangement.IsEntitled).To(Equal(true), "smart_management.is_entitled expected to be true")
		})
	})

	Context("When the Subs API says we *dont* have Smart Managment", func() {
		It("should give back a valid EntitlementsResponse with smart_management false", func() {
			rr, body := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, []string{}))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartMangement.IsEntitled).To(Equal(false), "smart_management.is_entitled expected to be false")
		})
	})

	Context("When the Subs API sends gback invalid data", func() {
		It("should give back a valid EntitlementsResponse with smart_management false", func() {
			rr, body := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, []string{}))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartMangement.IsEntitled).To(Equal(false), "smart_management.is_entitled expected to be false")
		})
	})
})
