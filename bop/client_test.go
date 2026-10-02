package bop

import (
	"encoding/json"

	"github.com/RedHatInsights/entitlements-api-go/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BOP Client", func() {

	Context("When passed a userName", func() {
		It("should build a valid request body", func() {
			inputName := "testuser"
			expected := userRequest{
				Users: []string{inputName},
			}

			outputBytes, err := makeRequestBody(inputName)
			Expect(err).To(BeNil())
			var actualRequest userRequest
			Expect(json.Unmarshal(outputBytes.Bytes(), &actualRequest)).To(BeNil())
			Expect(actualRequest).To(Equal(expected))
		})
	})

	Context("When passed a userName and url", func() {
		It("should construct a request object", func() {
			req, err := makeRequest("testuser", "fakeurl.com")
			Expect(err).To(BeNil())
			Expect(req.Header.Get("Content-Type")).To(Equal("application/json"))
		})
	})
})

var _ = Describe("validateBOPSettings", func() {

	Context("When all three params are provided", func() {
		It("should return nil", func() {
			err := validateBOPSettings("my-client-id", "my-token", "https://bop.example.com")
			Expect(err).To(BeNil())
		})
	})

	Context("When only clientId is missing", func() {
		It("should return an error mentioning BOP_CLIENT_ID", func() {
			err := validateBOPSettings("", "my-token", "https://bop.example.com")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPClientID))
			Expect(err.Error()).ToNot(ContainSubstring(config.Keys.BOPToken))
			Expect(err.Error()).ToNot(ContainSubstring(config.Keys.BOPURL))
		})
	})

	Context("When only token is missing", func() {
		It("should return an error mentioning BOP_TOKEN", func() {
			err := validateBOPSettings("my-client-id", "", "https://bop.example.com")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPToken))
			Expect(err.Error()).ToNot(ContainSubstring(config.Keys.BOPClientID))
			Expect(err.Error()).ToNot(ContainSubstring(config.Keys.BOPURL))
		})
	})

	Context("When only url is missing", func() {
		It("should return an error mentioning BOP_URL", func() {
			err := validateBOPSettings("my-client-id", "my-token", "")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPURL))
			Expect(err.Error()).ToNot(ContainSubstring(config.Keys.BOPClientID))
			Expect(err.Error()).ToNot(ContainSubstring(config.Keys.BOPToken))
		})
	})

	Context("When multiple params are missing", func() {
		It("should return an error listing clientId and token", func() {
			err := validateBOPSettings("", "", "https://bop.example.com")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPClientID))
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPToken))
			Expect(err.Error()).ToNot(ContainSubstring(config.Keys.BOPURL))
		})

		It("should return an error listing clientId and url", func() {
			err := validateBOPSettings("", "my-token", "")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPClientID))
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPURL))
			Expect(err.Error()).ToNot(ContainSubstring(config.Keys.BOPToken))
		})

		It("should return an error listing token and url", func() {
			err := validateBOPSettings("my-client-id", "", "")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPToken))
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPURL))
			Expect(err.Error()).ToNot(ContainSubstring(config.Keys.BOPClientID))
		})
	})

	Context("When all params are empty", func() {
		It("should return an error listing all three keys", func() {
			err := validateBOPSettings("", "", "")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPClientID))
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPToken))
			Expect(err.Error()).To(ContainSubstring(config.Keys.BOPURL))
		})
	})
})
