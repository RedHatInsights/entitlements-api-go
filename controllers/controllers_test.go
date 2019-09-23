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
const DEFAULT_ACCOUNT_NUMBER string = "540155"

func testRequest(method string, path string, accnum string, orgid string, fakeCaller func(string) SubscriptionsResponse) (*httptest.ResponseRecorder, EntitlementsResponse, string) {
	req, err := http.NewRequest(method, path, nil)
	Expect(err).To(BeNil(), "NewRequest error was not nil")

	ctx := context.Background()
	ctx = context.WithValue(ctx, identity.Key, identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: accnum,
			Internal: identity.Internal{
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
	return testRequest(method, path, DEFAULT_ACCOUNT_NUMBER, DEFAULT_ORG_ID, fakeCaller)
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
		testRequest("GET", "/", DEFAULT_ACCOUNT_NUMBER, "540155", fakeGetSubscriptions("540155", fakeResponse))
		testRequest("GET", "/", DEFAULT_ACCOUNT_NUMBER, "deadbeef12", fakeGetSubscriptions("deadbeef12", fakeResponse))
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

	Context("When the Subs API says we have Smart Management", func() {
		It("should give back a valid EntitlementsResponse with smart_management true", func() {
			fakeResponse := SubscriptionsResponse{
				StatusCode: 200,
				Data:       []string{"foo", "bar", "SVC3124", "SVC3851", "MCT3691"},
				CacheHit:   false,
			}

			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartManagement.IsEntitled).To(Equal(true))
			Expect(body.Ansible.IsEntitled).To(Equal(true))
			Expect(body.Migrations.IsEntitled).To(Equal(true))
		})
	})

	Context("When the Subs API says we have Smart Management (CMSfR SKU)", func() {
		It("should give back a valid EntitlementsResponse with smart_management true", func() {
			fakeResponse := SubscriptionsResponse{
				StatusCode: 200,
				Data:       []string{"foo", "bar", "SVC3851", "RH00068", "MCT3692"},
				CacheHit:   false,
			}

			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartManagement.IsEntitled).To(Equal(true))
			Expect(body.Ansible.IsEntitled).To(Equal(true))
			Expect(body.Migrations.IsEntitled).To(Equal(true))
		})
	})

	Context("When the Subs API says we *dont* have Smart Management", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       []string{"SVC3852", "MCT3693"},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with smart_management false", func() {
			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartManagement.IsEntitled).To(Equal(false))
			Expect(body.Ansible.IsEntitled).To(Equal(true))
			Expect(body.Migrations.IsEntitled).To(Equal(true))
		})
	})

	Context("When the account number is -1 or '' ", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       []string{"foo", "bar", "SVC3852", "SVC3124", "MCT3694"},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with insights, ansible and migrations false", func() {
			// testing with account number "-1"
			rr, body, _ := testRequest("GET", "/", "-1", DEFAULT_ORG_ID, fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(false))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartManagement.IsEntitled).To(Equal(true))
			Expect(body.Ansible.IsEntitled).To(Equal(false))
			Expect(body.Migrations.IsEntitled).To(Equal(false))
		})

		It("should give back a valid EntitlementsResponse with insights, ansible and migrations false", func() {
			// testing with account number ""
			rr, body, _ := testRequest("GET", "/", "", DEFAULT_ORG_ID, fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(false))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartManagement.IsEntitled).To(Equal(true))
			Expect(body.Ansible.IsEntitled).To(Equal(false))
			Expect(body.Migrations.IsEntitled).To(Equal(false))
		})

	})

	Context("When the Subs API says we have Ansible", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       []string{"SVC3852", "MCT3695", "MCT3696"},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with ansible true", func() {
			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartManagement.IsEntitled).To(Equal(false))
			Expect(body.Ansible.IsEntitled).To(Equal(true))
			Expect(body.Migrations.IsEntitled).To(Equal(true))
		})
	})

	Context("When the Subs API says we *dont* have Ansible", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       []string{"SVC3852"},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with ansible false", func() {
			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartManagement.IsEntitled).To(Equal(false))
			Expect(body.Ansible.IsEntitled).To(Equal(true))
			Expect(body.Migrations.IsEntitled).To(Equal(true))
		})
	})

	Context("When the Subs API says we have Migrations", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       []string{},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with migrations true", func() {
			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
			expectPass(rr.Result())
			Expect(body.Insights.IsEntitled).To(Equal(true))
			Expect(body.Openshift.IsEntitled).To(Equal(true))
			Expect(body.HybridCloud.IsEntitled).To(Equal(true))
			Expect(body.SmartManagement.IsEntitled).To(Equal(false))
			Expect(body.Ansible.IsEntitled).To(Equal(true))
			Expect(body.Migrations.IsEntitled).To(Equal(true))
		})
	})

	//tests disabled while not enforcing entitlements for hybrid cloud
	/*
		Context("When the Subs API says we have Hybrid", func() {
			It("should give back a valid EntitlementsResponse with all is_entitled true", func() {
				fakeResponse := SubscriptionsResponse{
					StatusCode: 200,
					Data:       []string{"SVCSER0566", "SVC3124", "SVC3852"},
					CacheHit:   false,
				}

				rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
				expectPass(rr.Result())
				Expect(body.Insights.IsEntitled).To(Equal(true))
				Expect(body.Openshift.IsEntitled).To(Equal(true))
				Expect(body.HybridCloud.IsEntitled).To(Equal(true))
				Expect(body.SmartMangement.IsEntitled).To(Equal(true))
			})
		})

		Context("When the Subs API says we *dont* have Hybrid", func() {
			It("should give back a valid EntitlementsResponse with hybrid is_entitled false", func() {
				fakeResponse := SubscriptionsResponse{
					StatusCode: 200,
					Data:       []string{"SVC1234", "SVC3124", "SVC5678"},
					CacheHit:   false,
				}

				rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, fakeResponse))
				expectPass(rr.Result())
				Expect(body.Insights.IsEntitled).To(Equal(true))
				Expect(body.Openshift.IsEntitled).To(Equal(true))
				Expect(body.HybridCloud.IsEntitled).To(Equal(false))
				Expect(body.SmartMangement.IsEntitled).To(Equal(true))
			})
		})
	*/

})
