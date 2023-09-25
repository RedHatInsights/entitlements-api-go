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
		It("should add an IN clause to the query", func () {
			query := builder.In("foo.bar", []string{"Cow", "Chicken"}).Build()
			Expect(query).To(Equal("foo.bar IN ('Cow','Chicken')"))
		})
	})

	When("And is called", func () {
		It("should add an AND clause to the query", func () {
			query := builder.
				In("foo.bar", []string{"some status"}).
				And().
				Equals("baz", "thonk").
				Build()
			Expect(query).To(Equal("foo.bar IN ('some status') AND baz = 'thonk'"))
		})
	})

	When("all operators are called", func () {
		It("correctly construct the query", func () {
			query := builder.
				Like("field", "nothonk").
				And().
				In("foo.bar", []string{"some status"}).
				And().
				Equals("baz", "thonk").
				Build()
			Expect(query).To(Equal("field LIKE 'nothonk' AND foo.bar IN ('some status') AND baz = 'thonk'"))
		})
	})
})