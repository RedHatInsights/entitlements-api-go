package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/RedHatInsights/entitlements-api-go/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

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

type opt func(reqStruct)

func MakeRequest(method, path string, options ...opt) *http.Request {
	r := reqStruct{}
	r.ID = identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: DEFAULT_ACCOUNT_NUMBER,
			User: identity.User{
				Internal: DEFAULT_IS_INTERNAL,
				Email:    DEFAULT_EMAIL,
			},
			Internal: identity.Internal{
				OrgID: DEFAULT_ORG_ID,
			},
		},
	}

	for _, o := range options {
		o(r)
	}

	r.Ctx = context.WithValue(context.Background(), identity.Key, r.ID)

	req, err := http.NewRequestWithContext(r.Ctx, r.Method, r.Path, nil)
	Expect(err).To(BeNil(), "NewRequest error was  not nil")
	return req

}

var _ = Describe("Seat Management", func() {

	var seatApi *SeatManagerApi

	BeforeEach(func() {
		seatApi = NewMockSeatManagerApi()
	})

	It("should return a list", func() {
		req := MakeRequest("GET", "/api/entitlements/v1/seats")
		rr := httptest.NewRecorder()
		seatApi.GetSeats(rr, req, api.GetSeatsParams{})

		Expect(rr.Result().StatusCode).To(Equal(200))
		Expect(rr.Result().Header.Get("Content-Type")).To(Equal("application/json"))

		var result api.ListSeatsResponsePagination
		json.NewDecoder(rr.Result().Body).Decode(&result)

		Expect(*result.Meta.Count).To(Equal(int64(1)))
		Expect(*result.Data[0].AccountUsername).To(Equal("testuser"))

	})
})
