package controllers

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/RedHatInsights/entitlements-api-go/config"
)

// BundleInfo provides Bundle names and SKUs
func BundleInfo() {
	bundlesYamlPath := config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml)
	bundlesYaml, err := ioutil.ReadFile(bundlesYamlPath)

	if err != nil {
		log.Panic(err)
		return
	}

	fmt.Println(bundlesYaml)
}
