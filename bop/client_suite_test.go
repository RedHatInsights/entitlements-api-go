package bop

import (
	"testing"

	. "github.com/RedHatInsights/entitlements-api-go/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBop(t *testing.T) {
	InitLogger()
	RegisterFailHandler(Fail)
	RunSpecs(t, "BOP Suite")
}
