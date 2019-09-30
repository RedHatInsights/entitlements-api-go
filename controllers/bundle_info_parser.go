package controllers

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/types"

	"gopkg.in/yaml.v2"
)

// BundleInfo provides Bundle names and SKUs
func BundleInfo() {
	yamlPath := config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml)
	bundlesYaml, err := ioutil.ReadFile(yamlPath)

	if err != nil {
		log.Panic(err)
		return
	}

	y := types.BundleDetails{}
	err = yaml.Unmarshal([]byte(bundlesYaml), &y)
    if err != nil {
        log.Fatalf("error: %v", err)
    }
	fmt.Println("Name: ", y)
}
