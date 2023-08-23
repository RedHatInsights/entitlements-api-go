package ams

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Query Builder Test", func () {
	var builder *QueryBuilder
	BeforeEach(func () {
		builder = NewQueryBuilder()
	})

	When("NewQueryBuilder is called", func () {
		It("should return a new instance of the builder", func () {
			Expect(builder).ToNot(BeIdenticalTo(NewQueryBuilder()))
		})
	})

	When("Like is called", func () {
		It("should add a LIKE clause to the query", func () {
			query := builder.Like("foo.bar", "baz").Build()
			Expect(query).To(Equal("foo.bar LIKE 'baz'"))
		})
	})

	When("Equals is called", func () {
		It("should add an EQUALS clause to the query", func () {
			query := builder.Equals("foo.bar", "baz").Build()
			Expect(query).To(Equal("foo.bar = 'baz'"))
		})
	})

	When("In is called", func () {
		It("should add an IN clause to the query and standardize casing", func () {
			query := builder.In("foo.bar", []string{"ALLCAPS", "ChIcKeN", "allLower"}).Build()
			Expect(query).To(Equal("foo.bar IN ('Allcaps','Chicken','Alllower')"))
		})
	})

	When("And is called", func () {
		It("should add an AND clause to the query", func () {
			query := builder.
				In("foo.bar", []string{"some status"}).
				And().
				Equals("baz", "thonk").
				Build()
			Expect(query).To(Equal("foo.bar IN ('Some Status') AND baz = 'thonk'"))
		})
	})
})