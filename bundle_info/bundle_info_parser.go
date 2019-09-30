package bundles

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/RedHatInsights/entitlements-api-go/config"
)

// BundleInfo provides Bundle names and SKUs
func BundleInfo() {
	specFilePath := config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml)
	specFile, err := ioutil.ReadFile(specFilePath)

	if err != nil {
		log.Panic(err)
		return
	}

	fmt.Println(specFile)
}
