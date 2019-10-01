package controllers

import (
	"io/ioutil"
	"log"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/types"

	"gopkg.in/yaml.v2"
)

// BundleInfo provides Bundle names and SKUs
func BundleInfo() []types.Bundle {
	yamlPath := config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml)
	bundlesYaml, err := ioutil.ReadFile(yamlPath)

	var bundles []types.Bundle

	if err != nil {
		log.Panic(err)
		return bundles
	}

	err = yaml.Unmarshal([]byte(bundlesYaml), &bundles)
	if err != nil {
		log.Fatalf("error: %+v", err)
	}

	return bundles
}
