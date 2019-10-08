package controllers

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/RedHatInsights/platform-go-middlewares/identity"
)

const DEFAULT_ORG_ID string = "4384938490324"
const DEFAULT_ACCOUNT_NUMBER string = "540155"

// func TestMain(m *testing.M) {
// 	bundleInfo = fakeBundleInfo()
// }

func init() {
	fmt.Println("Opening in test")
	bundleInfo = fakeBundleInfo()
}

func testRequest(method string, path string, accnum string, orgid string, fakeCaller func(string, string) SubscriptionsResponse, fakeBundleInfo []Bundle) (*httptest.ResponseRecorder, map[string]EntitlementsSection, string) {
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

	//GetBundleInfo = fakeGetBundle
	GetSubscriptions = fakeCaller

	Index()(rr, req)

	out, err := ioutil.ReadAll(rr.Result().Body)
	Expect(err).To(BeNil(), "ioutil.ReadAll error was not nil")

	rr.Result().Body.Close()

	var ret map[string]EntitlementsSection
	json.Unmarshal(out, &ret)

	return rr, ret, string(out)
}

func testRequestWithDefaultOrgId(method string, path string, fakeCaller func(string, string) SubscriptionsResponse, fakeBundleInfo []Bundle) (*httptest.ResponseRecorder, map[string]EntitlementsSection, string) {
	return testRequest(method, path, DEFAULT_ACCOUNT_NUMBER, DEFAULT_ORG_ID, fakeCaller, fakeBundleInfo)
}

func fakeGetSubscriptions(expectedOrgID string, expectedSkus string, response SubscriptionsResponse) func(string, string) SubscriptionsResponse {
	return func(orgID string, skus string) SubscriptionsResponse {
		Expect(expectedOrgID).To(Equal(orgID))
		return response
	}
}

func fakeBundleInfo() []Bundle {
	fakeBundle1 := Bundle{
		Name: "TestBundle1",
		Skus: []string{"SVC123", "SVC456", "MCT789"},
	}
	fakeBundle2 := Bundle{
		Name: "TestBundle2",
		Skus: []string{"MCT1122", "SVC3344"},
	}
	fakeBundle3 := Bundle{
		Name:           "TestBundle3",
		UseValidAccNum: false,
	}
	fakeBundle4 := Bundle{
		Name:           "TestBundle4",
		UseValidAccNum: true,
	}

	fakeBundles := []Bundle{fakeBundle1, fakeBundle2, fakeBundle3, fakeBundle4}

	return fakeBundles
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
		testRequest("GET", "/", DEFAULT_ACCOUNT_NUMBER, "540155", fakeGetSubscriptions("540155", "", fakeResponse), fakeBundleInfo())
		testRequest("GET", "/", DEFAULT_ACCOUNT_NUMBER, "deadbeef12", fakeGetSubscriptions("deadbeef12", "", fakeResponse), fakeBundleInfo())
	})

	Context("When the Subs API sends back an error", func() {
		It("should fail the response", func() {
			rr, _, rawBody := testRequestWithDefaultOrgId("GET", "/", func(string, string) SubscriptionsResponse {
				return SubscriptionsResponse{StatusCode: 500, Data: nil, CacheHit: false}
			}, fakeBundleInfo())

			Expect(rr.Result().StatusCode).To(Equal(500))
			Expect(rawBody).To(ContainSubstring(http.StatusText(500)))
		})
	})

	Context("When the bundles.yml isn't available", func() {
		It("should fail the response", func() {
			fakeResponse := SubscriptionsResponse{
				StatusCode: 200,
				Data:       []string{"foo", "bar"},
				CacheHit:   false,
			}

			rr, _, rawBody := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions("540155", "", fakeResponse), []Bundle{
				Bundle{Error: errors.New("bundles.yml unavailable")}})
			Expect(rr.Result().StatusCode).To(Equal(500))
			Expect(rawBody).To(ContainSubstring(http.StatusText(500)))
		})
	})

	Context("When the Subs API says we have SKUs to a Bundle", func() {
		It("should give back a valid EntitlementsResponse with that bundle true", func() {
			fakeResponse := SubscriptionsResponse{
				StatusCode: 200,
				Data:       []string{"SVC123", "SVC3851", "MCT3691"},
				CacheHit:   false,
			}

			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, "SVC3124,MCT3691", fakeResponse), fakeBundleInfo())
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(true))
		})
	})

	Context("When the Subs API says we *dont* have SKUs to a Bundle", func() {
		It("should give back a valid EntitlementsResponse with that bundle true", func() {
			fakeResponse := SubscriptionsResponse{
				StatusCode: 200,
				Data:       []string{"SVC123", "SVC3851", "MCT3691"},
				CacheHit:   false,
			}

			rr, body, _ := testRequestWithDefaultOrgId("GET", "/", fakeGetSubscriptions(DEFAULT_ORG_ID, "SVC3124,MCT3691", fakeResponse), fakeBundleInfo())
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(false))
		})
	})

	Context("When the account number is -1 or '' ", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       []string{"SVC123", "MCT1122", "SVC7788"},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with bundles using Valid Account Number false", func() {
			// testing with account number "-1"
			rr, body, _ := testRequest("GET", "/", "-1", DEFAULT_ORG_ID, fakeGetSubscriptions(DEFAULT_ORG_ID, "test", fakeResponse), fakeBundleInfo())
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle3"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle4"].IsEntitled).To(Equal(false))
		})

		It("should give back a valid EntitlementsResponse with bundles using Valid Account Number false", func() {
			// testing with account number ""
			rr, body, _ := testRequest("GET", "/", "", DEFAULT_ORG_ID, fakeGetSubscriptions(DEFAULT_ORG_ID, "test", fakeResponse), fakeBundleInfo())
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle3"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle4"].IsEntitled).To(Equal(false))
		})

	})

	Context("When a bundle uses only Valid Account Number", func() {
		fakeResponse := SubscriptionsResponse{
			StatusCode: 200,
			Data:       []string{},
			CacheHit:   false,
		}

		It("should give back a valid EntitlementsResponse with that bundle true", func() {
			// testing with account number ""
			rr, body, _ := testRequest("GET", "/", "123456", DEFAULT_ORG_ID, fakeGetSubscriptions(DEFAULT_ORG_ID, "test", fakeResponse), fakeBundleInfo())
			expectPass(rr.Result())
			Expect(body["TestBundle1"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle2"].IsEntitled).To(Equal(false))
			Expect(body["TestBundle3"].IsEntitled).To(Equal(true))
			Expect(body["TestBundle4"].IsEntitled).To(Equal(true))
		})

	})

})
