package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func readResponse(respBody io.ReadCloser) []byte {
	out, err := ioutil.ReadAll(respBody)
	Expect(err).To(BeNil(), "ioutil.ReadAll error was not nil")
	respBody.Close()

	return out
}

var _ = Describe("", func() {
	Context("When the request cannot be read", func() {
		It("should return an error and status 400", func() {
			// given
			req := httptest.NewRequest(http.MethodPost, "/foo", errReader(0))
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
			Expect(errorResp.Error.Message).To(ContainSubstring("test error"))
			Expect(errorResp.Error.Status).To(Equal(http.StatusBadRequest))
		})
	})

	Context("When the request to compliance service cannot be created", func() {
		It("should return an error and status 500", func() {
			// given
			config.GetConfig().Options.Set(config.Keys.ComplianceHost, "bad url that will cause an error in http.NewRequest\n")
			reqBody, _ := json.Marshal(types.ComplianceScreeningRequest{
				User: types.User{
					Id: "1234",
				},
				Account: types.Account{
					Primary: false,
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/foo", bytes.NewBuffer(reqBody))
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
			reqBody, _ := json.Marshal(types.ComplianceScreeningRequest{
				User: types.User{
					Id: "1234",
				},
				Account: types.Account{
					Primary: false,
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/foo", bytes.NewBuffer(reqBody))
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
			reqBody, _ := json.Marshal(types.ComplianceScreeningRequest{
				User: types.User{
					Id: "1234",
				},
				Account: types.Account{
					Primary: false,
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/foo", bytes.NewBuffer(reqBody))
			rr := httptest.NewRecorder()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == screeningPathV1 {
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
})
