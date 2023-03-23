package ocm

import (
	"os"
	"testing"

	"github.com/redhatinsights/mbop/internal/config"

	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/stretchr/testify/suite"
)

type OcmImplTestSuite struct {
	suite.Suite
	IsInternalLabel string
}

func (suite *OcmImplTestSuite) SetupSuite() {
	suite.IsInternalLabel = "internalLabelKey"
}

func (suite *OcmImplTestSuite) SetupTest() {
	config.Reset()
}

func (suite *OcmImplTestSuite) TestGetIsInternalMatch() {
	os.Setenv("IS_INTERNAL_LABEL", "internalLabelKey")
	l := &v1.LabelBuilder{}
	l.Key(suite.IsInternalLabel)
	l.Value("true")
	acctB := &v1.AccountBuilder{}
	acctB.Labels(l)
	acct, _ := acctB.Build()
	suite.Equal(true, getIsInternal(acct))
}

func (suite *OcmImplTestSuite) TestGetIsInternaEmptyLabels() {
	os.Setenv("IS_INTERNAL_LABEL", "internalLabelKey")
	acctB := &v1.AccountBuilder{}
	acct, _ := acctB.Build()
	suite.Equal(false, getIsInternal(acct))
}

func (suite *OcmImplTestSuite) TestGetIsInternalNoKeyMatch() {
	os.Setenv("IS_INTERNAL_LABEL", "foo")
	l := &v1.LabelBuilder{}
	l.Key(suite.IsInternalLabel)
	l.Value("true")
	acctB := &v1.AccountBuilder{}
	acctB.Labels(l)
	acct, _ := acctB.Build()
	suite.Equal(false, getIsInternal(acct))
}

func (suite *OcmImplTestSuite) TestGetIsInternalNoValMatch() {
	os.Setenv("IS_INTERNAL_LABEL", "internalLabelKey")
	l := &v1.LabelBuilder{}
	l.Key(suite.IsInternalLabel)
	l.Value("false")
	acctB := &v1.AccountBuilder{}
	acctB.Labels(l)
	acct, _ := acctB.Build()
	suite.Equal(false, getIsInternal(acct))
}

func TestOcmImp(t *testing.T) {
	suite.Run(t, new(OcmImplTestSuite))
}
