package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("http client", func() {
	It("should not create a new client every call", func(){
		// given
		client1 := getClient()

		// when
		client2 := getClient()

		// then
		Expect(client1).To(BeIdenticalTo(client2))
	})
})