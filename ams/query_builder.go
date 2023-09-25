package ams

import (
	"strings"

	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/sirupsen/logrus"
)

// QueryBuilder is a struct that contains methods to build a query to send to AMS.
// These queries are built in a SQL like way, in that they use SQL operators.
// For example, a query can look like this: "plan.id LIKE 'WISDOM' AND status IN ('Active')"
// These methods construct a query string by operator, accepting a field and value to use.
type QueryBuilder struct {
	query	string
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: "",
	}
}

var parentheses 	= "()"
var singleQuotes 	= "''"

func (builder *QueryBuilder) queryOperator(operator, field, value, valWrappers string) {
	if len(valWrappers) < 2 {
		valWrappers = singleQuotes
	}

	var str strings.Builder
	str.WriteString(builder.query)
	str.WriteString(field)
	str.WriteString(" ")
	str.WriteString(operator)
	str.WriteString(" ")
	str.WriteString(string(valWrappers[0]))
	str.WriteString(value)
	str.WriteString(string(valWrappers[1]))
	
	builder.query = str.String()
}

func (builder *QueryBuilder) Like(field, value string) *QueryBuilder {
	builder.queryOperator("LIKE", field, value, singleQuotes)
	
	return builder
}

func (builder *QueryBuilder) Equals(field, value string) *QueryBuilder {
	builder.queryOperator("=", field, value, singleQuotes)

	return builder
}

func (builder *QueryBuilder) In(field string, values []string) *QueryBuilder {
	var value strings.Builder
	for index,element := range values {
		value.WriteString("'")
		value.WriteString(element)
		value.WriteString("'")
		if index < len(values)-1 {
			value.WriteString(",")
		}
	}

	builder.queryOperator("IN", field, value.String(), parentheses)
	
	return builder
}

func (builder *QueryBuilder) And() *QueryBuilder {
	var str strings.Builder
	str.WriteString(builder.query)
	str.WriteString(" AND ")

	builder.query = str.String()
	
	return builder
}

func (builder *QueryBuilder) Build() string {
	logger.Log.WithFields(logrus.Fields{"ams_search_query":builder.query}).Debug("built ams search query")
	return builder.query
}