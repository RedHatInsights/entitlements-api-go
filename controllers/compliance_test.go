package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

var defaultEmail = "test@redhat.com"

func readResponse(respBody io.ReadCloser) []byte {
	out, err := io.ReadAll(respBody)
	Expect(err).To(BeNil(), "io.ReadAll error was not nil")
	respBody.Close()

	return out
}

func getContextWithIdentity(username string) context.Context {
	ctx := context.Background()
	ctx = identity.WithIdentity(ctx, identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: "540155",
			User: &identity.User{
				Username: username,
			},
		},
	})

	return ctx
}

func getContextWithServiceAccount() context.Context {
	ctx := context.Background()
	ctx = identity.WithIdentity(ctx, identity.XRHID{
		Identity: identity.Identity{
			AccountNumber: "540155",
			Type:          "ServiceAccount",
			User:          nil,
			ServiceAccount: &identity.ServiceAccount{
				Username: "service-account-test",
				ClientId: "test-client-id",
			},
			Internal: identity.Internal{
				OrgID: "11789772",
			},
		},
	})

	return ctx
}

var _ = Describe("Compliance Controller", func() {
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

	Context("When the request to compliance service fails due to timeout", func() {
		It("should return a specific error and status 500", func() {
			// given
			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req = req.WithContext(getContextWithIdentity(defaultEmail))
			rr := httptest.NewRecorder()

			cfg := config.GetConfig().Options
			wait := cfg.GetInt(config.Keys.ITServicesTimeoutSeconds) + 1

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == cfg.GetString(config.Keys.CompAPIBasePath) {
					time.Sleep(time.Duration(wait) * time.Second)
				}
			}))
			defer server.Close()
			cfg.Set(config.Keys.ComplianceHost, server.URL)

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
			Expect(errorResp.Error.Message).To(ContainSubstring("Request to Export Compliance Service timed out"))
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
			defer server.Close()
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

	Context("When the request to compliance service is successful and error from compliance", func() {
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
			defer server.Close()
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

	Context("When identity is a Service Account", func() {
		It("should reject Service Account with 400 and clear error message", func() {
			// given
			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req = req.WithContext(getContextWithServiceAccount())
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
			Expect(errorResp.Error.Message).To(ContainSubstring("Service Accounts are not supported"))
			Expect(errorResp.Error.Status).To(Equal(http.StatusBadRequest))
		})
	})
})
