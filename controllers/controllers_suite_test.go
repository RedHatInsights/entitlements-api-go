package controllers

import (
	"testing"

	. "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestControllers(t *testing.T) {
	InitLogger()
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Controllers Suite")
}
