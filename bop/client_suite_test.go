package bop

import (
	"testing"

	"github.com/RedHatInsights/entitlements-api-go/config"
	. "github.com/RedHatInsights/entitlements-api-go/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBop(t *testing.T) {
	InitLogger()
	RegisterFailHandler(Fail)
	if config.GetConfig().Options.GetBool(config.Keys.DisableSeatManager) {
		GinkgoWriter.Println("Seats apis are disabled... skipping bop test suite")
		return
	}
	RunSpecs(t, "BOP Suite")
}
