package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/RedHatInsights/entitlements-api-go/ams"
	"github.com/RedHatInsights/entitlements-api-go/api"
	"github.com/RedHatInsights/entitlements-api-go/bop"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

const DEFAULT_ORG_ADMIN = true

type reqStruct struct {
	Method     string
	Path       string
	Account    string
	IsInternal bool
	Email      string
	OrgId      string
	Ctx        context.Context
	ID         identity.XRHID
}

type opt func(*reqStruct)

func OrgAdmin(orgAdmin bool) opt {
	return func(o *reqStruct) {
		o.ID.Identity.User.OrgAdmin = orgAdmin
	}
}

func OrgId(orgId string) opt {
	return func(o *reqStruct) {
		o.ID.Identity.Internal.OrgID = orgId
	}
}

func MakeRequest(method, path string, body io.Reader, options ...opt) *http.Request {
	r := reqStruct{}
	r.ID = identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: DEFAULT_ACCOUNT_NUMBER,
			User: identity.User{
				Internal: DEFAULT_IS_INTERNAL,
				Email:    DEFAULT_EMAIL,
				OrgAdmin: DEFAULT_ORG_ADMIN,
			},
			Internal: identity.Internal{
				OrgID: DEFAULT_ORG_ID,
			},
		},
	}

	for _, o := range options {
		o(&r)
	}

	r.Ctx = context.WithValue(context.Background(), identity.Key, r.ID)

	req, err := http.NewRequestWithContext(r.Ctx, r.Method, r.Path, body)
	Expect(err).To(BeNil(), "NewRequest error was  not nil")
	return req

}

var _ = Describe("using the seat managment api", func() {
	var client ams.AMSInterface
	var bopClient bop.Bop
	var seatApi *SeatManagerApi
	var rr *httptest.ResponseRecorder

	BeforeEach(func() {
		client = &ams.TestClient{}
		bopClient, _ = bop.NewClient(true)
		seatApi = NewSeatManagerApi(client, bopClient)
		rr = httptest.NewRecorder()
	})

	When("removing a user from a seat", func() {
		Context("and the caller is an org admin", func() {
			It("should remove the requested user's subscription", func() {
				req := MakeRequest("DELETE", "", nil)
				seatApi.DeleteSeatsId(rr, req, "1")
				Expect(rr.Result().StatusCode).To(Equal(http.StatusNoContent))
			})
		})
		Context("and the caller is not an org admin", func() {
			It("should deny the request", func() {
				req := MakeRequest("DELETE", "", nil, OrgAdmin(false))
				seatApi.DeleteSeatsId(rr, req, "1")
				Expect(rr.Result().StatusCode).To(Equal(http.StatusForbidden))
			})
		})
		Context("and the caller is in a different org from the target", func() {
			It("should deny the request", func() {
				req := MakeRequest("DELETE", "", nil, OrgId("12345"))
				seatApi.DeleteSeatsId(rr, req, "1")
				Expect(rr.Result().StatusCode).To(Equal(http.StatusForbidden))
			})
		})
		Context("and no subscription is found", func() {
			It("should cause an internal error", func() {
				req := MakeRequest("DELETE", "", nil)
				seatApi.DeleteSeatsId(rr, req, "")
				Expect(rr.Result().StatusCode).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	When("listing seats", func() {
		It("should return a list", func() {
			req := MakeRequest("GET", "/api/entitlements/v1/seats", nil)
			seatApi.GetSeats(rr, req, api.GetSeatsParams{})

			Expect(rr.Result().StatusCode).To(Equal(http.StatusOK))
			Expect(rr.Result().Header.Get("Content-Type")).To(Equal("application/json"))

			var result api.ListSeatsResponsePagination
			json.NewDecoder(rr.Result().Body).Decode(&result)

			Expect(*result.Meta.Count).To(Equal(int64(1)))
			Expect(*result.Data[0].AccountUsername).To(Equal("testuser"))

		})
		Context("and seats with active status is excluded", func() {
			It("should return an empty list since only one seat is created and its active", func() {
				req := MakeRequest("GET", "/api/entitlements/v1/seats", nil)
				seatApi.GetSeats(rr, req, api.GetSeatsParams{
					ExcludeStatus: &api.ExcludeStatus{"active"},
				})

				Expect(rr.Result().StatusCode).To(Equal(http.StatusOK))
				Expect(rr.Result().Header.Get("Content-Type")).To(Equal("application/json"))

				var result api.ListSeatsResponsePagination
				json.NewDecoder(rr.Result().Body).Decode(&result)

				Expect(*result.Meta.Count).To(Equal(int64(0)))
			})
		})
		Context("and exclude status is nil", func() {
			It("should return a list with all seats", func() {
				req := MakeRequest("GET", "/api/entitlements/v1/seats", nil)
				seatApi.GetSeats(rr, req, api.GetSeatsParams{
					ExcludeStatus: nil,
				})

				Expect(rr.Result().StatusCode).To(Equal(http.StatusOK))
				Expect(rr.Result().Header.Get("Content-Type")).To(Equal("application/json"))

				var result api.ListSeatsResponsePagination
				json.NewDecoder(rr.Result().Body).Decode(&result)

				Expect(*result.Meta.Count).To(Equal(int64(1)))
				Expect(*result.Data[0].AccountUsername).To(Equal("testuser"))
			})
		})
		Context("and exclude status is empty", func() {
			It("should return a list with all seats", func() {
				req := MakeRequest("GET", "/api/entitlements/v1/seats", nil)
				seatApi.GetSeats(rr, req, api.GetSeatsParams{
					ExcludeStatus: &api.ExcludeStatus{},
				})

				Expect(rr.Result().StatusCode).To(Equal(http.StatusOK))
				Expect(rr.Result().Header.Get("Content-Type")).To(Equal("application/json"))

				var result api.ListSeatsResponsePagination
				json.NewDecoder(rr.Result().Body).Decode(&result)

				Expect(*result.Meta.Count).To(Equal(int64(1)))
				Expect(*result.Data[0].AccountUsername).To(Equal("testuser"))
			})
		})
		Context("and limit is too small", func() {
			It("should return a bad request", func() {
				req := MakeRequest("GET", "/api/entitlements/v1/seats", nil)
				seatApi.GetSeats(rr, req, api.GetSeatsParams{
					Limit: toPtr(int(0)),
				})

				Expect(rr.Result().StatusCode).To(Equal(http.StatusBadRequest))
			})
		})
		Context("and offset is too small", func() {
			It("should return a bad request", func() {
				req := MakeRequest("GET", "/api/entitlements/v1/seats", nil)
				seatApi.GetSeats(rr, req, api.GetSeatsParams{
					Offset: toPtr(int(-1)),
				})

				Expect(rr.Result().StatusCode).To(Equal(http.StatusBadRequest))
			})
		})
	})

	When("adding a user to a seat", func() {
		Context("the caller is an org admin", func() {
			It("should return a 200", func() {
				b, err := json.Marshal(api.SeatRequest{
					AccountUsername: "test-user",
				})
				Expect(err).To(BeNil())

				req := MakeRequest("POST", "/api/entitlements/v1/seats", bytes.NewBuffer(b))
				seatApi.PostSeats(rr, req)

				Expect(rr.Result().StatusCode).To(Equal(200))
			})
		})

		Context("the caller is not an org admin", func() {
			It("should return a 403", func() {
				b, err := json.Marshal(api.SeatRequest{
					AccountUsername: "test-user",
				})
				Expect(err).To(BeNil())

				req := MakeRequest("POST", "/api/entitlements/v1/seats", bytes.NewBuffer(b), OrgAdmin(false))
				seatApi.PostSeats(rr, req)

				Expect(rr.Result().StatusCode).To(Equal(403))
			})
		})

		Context("the target is in a different org from the caller", func() {
			It("should not assign the user a seat", func() {
				mismatchApi := NewSeatManagerApi(client, &bop.Mock{
					OrgId: "12345",
				})
				b, err := json.Marshal(api.SeatRequest{
					AccountUsername: "test-user",
				})
				Expect(err).To(BeNil())

				req := MakeRequest("POST", "/api/entitlements/v1/seats", bytes.NewBuffer(b), OrgAdmin(false))
				mismatchApi.PostSeats(rr, req)

				Expect(rr.Result().StatusCode).To(Equal(403))
			})
		})
	})
})
