package controllers

import (
	"testing"

	. "github.com/RedHatInsights/entitlements-api-go/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestControllers(t *testing.T) {
	InitLogger()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controllers Suite")
}
