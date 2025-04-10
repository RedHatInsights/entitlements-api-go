package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	t "github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

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

	var currentSubs t.SubModel
	err = json.Unmarshal(data, &currentSubs)
	if err != nil {
		return t.SubModel{}, err
	}
	return currentSubs, nil
}

func getUpdates(cfg *viper.Viper) ([]t.Bundle, error) {
	bundlesYaml, err := os.ReadFile(cfg.GetString(config.Keys.BundleInfoYaml))
	if err != nil {
		return []t.Bundle{}, err
	}

	var m []t.Bundle
	err = yaml.Unmarshal(bundlesYaml, &m)
	if err != nil {
		return []t.Bundle{}, err
	}

	return m, nil

}

func postUpdates(cfg *viper.Viper, client *http.Client, data []byte) error {
	url := fmt.Sprintf("%s%s", cfg.GetString(config.Keys.SubsHost), cfg.GetString(config.Keys.SubAPIBasePath))
	req, err := client.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer req.Body.Close()

	return nil

}

func main() {
	c := config.GetConfig()
	client := getClient(c)
	options := c.Options
	runSync := options.GetBool(config.Keys.RunBundleSync)

	if !runSync {
		fmt.Println("Bundle sync disabled")
		return
	}

	endpoints := strings.Split(c.Options.GetString(config.Keys.Features), ",")
	for _, endpoint := range endpoints {
		skus := make(map[string][]string)
		current_skus := make(map[string][]string)
		url := fmt.Sprintf("%s%s",
			options.GetString(config.Keys.SubsHost),
			options.GetString(config.Keys.SubAPIBasePath))
		current, err := getCurrent(client, url+endpoint)
		if err != nil {
			log.Fatalf("Unable to get current subscriptions: %s", err)
			os.Exit(1)
		}

		sku_updates, err := getUpdates(options)
		if err != nil {
			log.Fatalf("Unable to get updated YAML: %s", err)
			os.Exit(1)
		}
		for _, v := range sku_updates {
			if v.Name == endpoint {
				skus[endpoint] = append(skus[endpoint], v.Skus...)
			}
		}

		for _, v := range current.Rules.MatchProducts {
			current_skus[endpoint] = append(current_skus[endpoint], v.SkuCodes...)
		}

		sort.Strings(skus[endpoint])
		sort.Strings(current_skus[endpoint])

		if assertEq(skus[endpoint], current_skus[endpoint]) {
			fmt.Printf("No updates for %s\n", endpoint)
		} else {
			var m []t.MatchProducts
			m = append(m, t.MatchProducts{SkuCodes: skus[endpoint]})
			v := t.SubModel{
				Name: endpoint,
				Rules: t.Rules{
					MatchProducts: m,
				},
			}
			b, err := json.Marshal(v)
			if err != nil {
				log.Fatalf("Failed to Marshal updated JSON: %s", err)
				os.Exit(1)
			}
			err = postUpdates(options, client, b)
			if err != nil {
				log.Fatalf("Unable to post updates to subscriptions API: %s", err)
				os.Exit(1)
			} else {
				fmt.Printf("Updated %s\n", endpoint)
			}
		}
	}
}
