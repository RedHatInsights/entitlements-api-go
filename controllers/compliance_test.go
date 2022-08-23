package controllers

import (
	"context"
	"encoding/json"
	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

var defaultEmail = "test@redhat.com"

func readResponse(respBody io.ReadCloser) []byte {
	out, err := ioutil.ReadAll(respBody)
	Expect(err).To(BeNil(), "ioutil.ReadAll error was not nil")
	respBody.Close()

	return out
}

func getContextWithIdentity(username string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, identity.Key, identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: "540155",
			User: identity.User{
				Username: username,
			},
		},
	})

	return ctx
}

var _ = Describe("", func() {
	Context("When username is empty", func() {
		It("should return an error and status 400", func() {
			// given
			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req = req.WithContext(getContextWithIdentity(""))
			rr := httptest.NewRecorder()

			// when
			Compliance()(rr, req)

			// then
			Expect(rr.Result().StatusCode).To(Equal(http.StatusBadRequest))
			resp := readResponse(rr.Result().Body)

			var errorResp types.RequestErrorResponse
			err := json.Unmarshal(resp, &errorResp)
			Expect(err).To(BeNil(), "Error unmarshalling server response")

			Expect(errorResp.Error).ToNot(BeNil())
			Expect(errorResp.Error.Message).To(ContainSubstring("x-rh-identity header has a missing or whitespace username"))
			Expect(errorResp.Error.Status).To(Equal(http.StatusBadRequest))
		})
	})

	Context("When username is whitespace", func() {
		It("should return an error and status 400", func() {
			// given
			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req = req.WithContext(getContextWithIdentity("           "))
			rr := httptest.NewRecorder()

			// when
			Compliance()(rr, req)

			// then
			Expect(rr.Result().StatusCode).To(Equal(http.StatusBadRequest))
			resp := readResponse(rr.Result().Body)

			var errorResp types.RequestErrorResponse
			err := json.Unmarshal(resp, &errorResp)
			Expect(err).To(BeNil(), "Error unmarshalling server response")

			Expect(errorResp.Error).ToNot(BeNil())
			Expect(errorResp.Error.Message).To(ContainSubstring("x-rh-identity header has a missing or whitespace username"))
			Expect(errorResp.Error.Status).To(Equal(http.StatusBadRequest))
		})
	})

	Context("When the request to compliance service cannot be created", func() {
		It("should return an error and status 500", func() {
			// given
			config.GetConfig().Options.Set(config.Keys.ComplianceHost, "bad url that will cause an error in http.NewRequest\n")
			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req = req.WithContext(getContextWithIdentity(defaultEmail))
			rr := httptest.NewRecorder()

			// when
			Compliance()(rr, req)

			// then
			Expect(rr.Result().StatusCode).To(Equal(http.StatusInternalServerError))
			resp := readResponse(rr.Result().Body)

			var errorResp types.DependencyErrorResponse
			err := json.Unmarshal(resp, &errorResp)
			Expect(err).To(BeNil(), "Error unmarshalling server response")

			Expect(errorResp.Error).ToNot(BeNil())
			Expect(errorResp.Error.Message).ToNot(BeNil())
			Expect(errorResp.Error.Message).To(ContainSubstring("Unexpected error while creating request to Export Compliance Service"))
			Expect(errorResp.Error.Status).To(Equal(http.StatusInternalServerError))
			Expect(errorResp.Error.Service).To(Equal(complianceServiceName))
		})
	})

	Context("When the request to compliance service fails", func() {
		It("should return an error and status 500", func() {
			// given
			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req = req.WithContext(getContextWithIdentity(defaultEmail))
			rr := httptest.NewRecorder()

			server := httptest.NewUnstartedServer(http.NotFoundHandler())
			config.GetConfig().Options.Set(config.Keys.ComplianceHost, server.URL)

			// when
			Compliance()(rr, req)

			// then
			Expect(rr.Result().StatusCode).To(Equal(http.StatusInternalServerError))
			resp := readResponse(rr.Result().Body)

			var errorResp types.DependencyErrorResponse
			err := json.Unmarshal(resp, &errorResp)
			Expect(err).To(BeNil(), "Error unmarshalling server response")

			Expect(errorResp.Error).ToNot(BeNil())
			Expect(errorResp.Error.Message).ToNot(BeNil())
			Expect(errorResp.Error.Message).To(ContainSubstring("Unexpected error returned on request to Export Compliance Service"))
			Expect(errorResp.Error.Status).To(Equal(http.StatusInternalServerError))
			Expect(errorResp.Error.Service).To(Equal(complianceServiceName))
		})
	})

	Context("When the request to compliance service is successful", func() {
		It("should return a body and successful status code", func() {
			// given
			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req = req.WithContext(getContextWithIdentity(defaultEmail))
			rr := httptest.NewRecorder()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == config.GetConfig().Options.GetString(config.Keys.CompAPIBasePath) {
					resp, _ := json.Marshal(types.ComplianceScreeningResponse{
						Result:      "OK",
						Description: "",
					})
					w.WriteHeader(http.StatusOK)
					w.Write(resp)
				}
			}))
			config.GetConfig().Options.Set(config.Keys.ComplianceHost, server.URL)

			// when
			Compliance()(rr, req)

			// then
			Expect(rr.Result().StatusCode).To(Equal(http.StatusOK))
			resp := readResponse(rr.Result().Body)

			var response types.ComplianceScreeningResponse
			err := json.Unmarshal(resp, &response)
			Expect(err).To(BeNil(), "Error unmarshalling server response")

			Expect(response.Result).To(Equal("OK"))
			Expect(response.Description).To(Equal(""))
		})
	})

	Context("When the request to compliance service is fails on error from compliance", func() {
		It("should return a body and appropriate status code", func() {
			// given
			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req = req.WithContext(getContextWithIdentity(defaultEmail))
			rr := httptest.NewRecorder()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == config.GetConfig().Options.GetString(config.Keys.CompAPIBasePath) {
					resp, _ := json.Marshal(types.ComplianceScreeningErrorResponse{
						Errors: []types.ComplianceScreeningError{
							{
								Error:        "no_such_user",
								IdentityType: "login",
								Identity:     defaultEmail,
							},
						},
					})
					w.WriteHeader(http.StatusBadRequest)
					w.Write(resp)
				}
			}))
			config.GetConfig().Options.Set(config.Keys.ComplianceHost, server.URL)

			// when
			Compliance()(rr, req)

			// then
			Expect(rr.Result().StatusCode).To(Equal(http.StatusBadRequest))
			resp := readResponse(rr.Result().Body)

			var response types.ComplianceScreeningErrorResponse
			err := json.Unmarshal(resp, &response)
			Expect(err).To(BeNil(), "Error unmarshalling server response")

			Expect(response.Errors).To(HaveLen(1))
			Expect(response.Errors).To(ContainElement(types.ComplianceScreeningError{
				Error:        "no_such_user",
				IdentityType: "login",
				Identity:     defaultEmail,
			}))
		})
	})
})
