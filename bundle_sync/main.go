package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	t "github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var dryRun bool

// assertEq compares two slices of strings and returns true if they are equal
func assertEq(test []string, ans []string) bool {
	return reflect.DeepEqual(test, ans)
}

// getClient sets up the http client for the subscriptions API
func getClient(cfg *config.EntitlementsConfig) *http.Client {

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*cfg.Certs},
		RootCAs:      cfg.RootCAs,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * 10,
	}

	return client
}

func getCurrent(client *http.Client, url string) (t.SubModel, error) {
	resp, err := client.Get(url)

	if resp.StatusCode == 404 {
		// since `postUpdates` is an upsert, this will allow us to add new features
		// into our config, and `getUpdates` will source the feature SKU list and
		// create the new feature when it's not found.
		return t.SubModel{}, nil
	}

	if err != nil {
		return t.SubModel{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return t.SubModel{}, err
	}
	
	log.Printf("raw response from '%s': '%s'", url, string(data))

	var currentSubs t.SubModel
	err = json.Unmarshal(data, &currentSubs)
	if err != nil {
		return t.SubModel{}, err
	}
	return currentSubs, nil
}

// filter out duplicate skus in a slice by converting to a set, then back to a slice
func uniqueSkus(skus []string) ([]string){
	uniqueSkusSet := make(map[string]struct{})
	for _, sku := range skus {
		uniqueSkusSet[sku] = struct{}{}
	}

	uniqueSkus := make([]string, 0, len(uniqueSkusSet))

	for sku,_ := range uniqueSkusSet {
		uniqueSkus = append(uniqueSkus, sku)
	}
	
	slices.Sort(uniqueSkus)
	return uniqueSkus
}

func getBundlesConfig(cfg *viper.Viper) (map[string]t.Bundle, error) {
	bundlesMap := make(map[string]t.Bundle)
	bundlesYaml, err := os.ReadFile(cfg.GetString(config.Keys.BundleInfoYaml))
	if err != nil {
		return bundlesMap, err
	}

	var bundles []t.Bundle
	err = yaml.Unmarshal(bundlesYaml, &bundles)
	if err != nil {
		return bundlesMap, err
	}

	paidFeatureSuffix := cfg.GetString(config.Keys.PaidFeatureSuffix)

	for _, bundle := range bundles {
		if bundle.IsPaid() {
			// expand this into 2 features: "feature" and "feature_paid"
			fullBundle := t.Bundle{
				Name: bundle.Name,
				Skus: uniqueSkus(append(bundle.EvalSkus, bundle.PaidSkus...)),
				UseValidAccNum: bundle.UseValidAccNum,
				UseValidOrgId: bundle.UseValidOrgId,
				UseIsInternal: bundle.UseIsInternal,
			}

			paidBundle := t.Bundle{
				Name: bundle.Name + paidFeatureSuffix,
				Skus: uniqueSkus(bundle.PaidSkus),
				UseValidAccNum: bundle.UseValidAccNum,
				UseValidOrgId: bundle.UseValidOrgId,
				UseIsInternal: bundle.UseIsInternal,
			}

			bundlesMap[fullBundle.Name] = fullBundle
			bundlesMap[paidBundle.Name] = paidBundle
		} else {
			bundlesMap[bundle.Name] = bundle
		}

	}

	return bundlesMap, nil

}

func postUpdates(cfg *viper.Viper, client *http.Client, data []byte) error {
	url := fmt.Sprintf("%s%s", cfg.GetString(config.Keys.SubsHost), cfg.GetString(config.Keys.SubAPIBasePath))

	if dryRun {
		// print updates that would be made but don't actually run them
		log.Printf("*** POST '%s' - '%s'", url, string(data))
		return nil
	}

	resp, err := client.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		respBody := string(body)
		return fmt.Errorf("error posting update -- response: '%s', status: '%d'. url: '%s', request body: '%s'", respBody, resp.StatusCode, url, string(data))
	}

	defer resp.Body.Close()

	return nil

}

func main() {
	flag.BoolVar(&dryRun, "dry-run", false, "Include to do a dry run which won't post updates and print the updates that would happen")
	flag.Parse()
	
	c := config.GetConfig()
	client := getClient(c)
	options := c.Options
	runSync := options.GetBool(config.Keys.RunBundleSync)

	if !runSync {
		fmt.Println("Bundle sync disabled")
		return
	}
	
	url := fmt.Sprintf("%s%s",
		options.GetString(config.Keys.SubsHost),
		options.GetString(config.Keys.SubAPIBasePath))

	bundlesConfig, err := getBundlesConfig(options)
	if err != nil {
		log.Fatalf("Unable to get updated YAML: %s", err)
		os.Exit(1)
	}

	endpoints := strings.Split(c.Options.GetString(config.Keys.Features), ",")
	var paidEndpoints []string

	// expand any endpoint into "feature" and "feature_paid" if it has paid skus configured
	paidFeatureSuffix := options.GetString(config.Keys.PaidFeatureSuffix)
	for _, endpoint := range endpoints {
		paidFeatureName := endpoint + paidFeatureSuffix
		_, exists := bundlesConfig[paidFeatureName]

		if exists {
			paidEndpoints = append(paidEndpoints, paidFeatureName)
		}
	}
	endpoints = append(endpoints, paidEndpoints...)

	for _, endpoint := range endpoints {
		log.Printf("Checking for updates to %s\n", endpoint)
		skus := make(map[string][]string)
		current_skus := make(map[string][]string)
		current, err := getCurrent(client, url+endpoint)
		if err != nil {
			log.Fatalf("Unable to get current features: %s", err)
			os.Exit(1)
		}

		bundle, exists := bundlesConfig[endpoint]
		if exists {
			// this shouln't ever be false, ie we should never have a feature listed that doesn't exist in the bundle config
			// but in case we do, maybe due to a typo or misinput, this check will prevent an error from stopping the program
			skus[endpoint] = append(skus[endpoint], bundle.Skus...)
		}

		for _, rule := range current.Rules {
			for _, mp := range rule.MatchProducts {
				current_skus[endpoint] = append(current_skus[endpoint], mp.SkuCodes...)
			}
		}

		sort.Strings(skus[endpoint])
		sort.Strings(current_skus[endpoint])

		if assertEq(skus[endpoint], current_skus[endpoint]) {
			fmt.Printf("No updates for %s\n", endpoint)
		} else {
			var m []t.MatchProducts
			m = append(m, t.MatchProducts{SkuCodes: skus[endpoint]})
			rules := t.Rules{MatchProducts: m,}
			v := t.SubModel{
				Name: endpoint,
				Rules: []t.Rules{rules},
			}
			b, err := json.Marshal(v)
			if err != nil {
				log.Fatalf("Failed to Marshal updated JSON: %s", err)
				os.Exit(1)
			}
			err = postUpdates(options, client, b)
			if err != nil {
				log.Fatalf("Unable to post updates to features API: %s", err)
				os.Exit(1)
			} else {
				fmt.Printf("Updated %s\n", endpoint)
			}
		}
	}
}
