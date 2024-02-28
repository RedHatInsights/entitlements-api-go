package ams

import (
	"testing"

	"github.com/RedHatInsights/entitlements-api-go/config"
	. "github.com/RedHatInsights/entitlements-api-go/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAms(t *testing.T) {
	InitLogger()
	RegisterFailHandler(Fail)
	if config.GetConfig().Options.GetBool(config.Keys.DisableSeatManager) {
		GinkgoWriter.Println("Seats apis are disabled... skipping ams suite")
		return
	}
	RunSpecs(t, "AMS Suite")
}