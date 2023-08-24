package ams

import (
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// QueryBuilder is a struct that contains methods to build a query to send to AMS.
// These queries are built in a SQL like way, in that they use SQL operators.
// For example, a query can look like this: "plan.id LIKE 'WISDOM' AND status IN ('Active')"
// These methods construct a query string by operator, accepting a field and value to use.
type QueryBuilder struct {
	query	string
	caser	cases.Caser
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: "",
		caser: cases.Title(language.Und),
	}
}

func (builder *QueryBuilder) queryOperator(operator, field, value string) {
	if operator == "IN" {
		builder.query += field + " " + operator + " " + "(" + value + ")"
	} else {
		builder.query += field + " " + operator + " " + "'" + value + "'"
	}
}

func (builder *QueryBuilder) Like(field, value string) *QueryBuilder {
	builder.queryOperator("LIKE", field, value)
	
	return builder
}

func (builder *QueryBuilder) Equals(field, value string) *QueryBuilder {
	builder.queryOperator("=", field, value)

	return builder
}

func (builder *QueryBuilder) In(field string, values []string) *QueryBuilder {
	value := ""
	for index,element := range values {
		// AMS is case sensitive, so standardize the input here by using Title Case
		value += "'" + builder.caser.String(element) + "'"
		if index < len(values)-1 {
			value += ","
		}
	}

	builder.queryOperator("IN", field, value)
	
	return builder
}

func (builder *QueryBuilder) And() *QueryBuilder {
	builder.query = builder.query + " AND "
	
	return builder
}

func (builder *QueryBuilder) Build() string {
	logger.Log.WithFields(logrus.Fields{"ams_search_query":builder.query}).Debug("built ams search query")
	return builder.query
}