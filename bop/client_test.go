package bop

import (
	"encoding/json"

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
