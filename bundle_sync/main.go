package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	cfg "github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	endpoints = []string{"ansible", "smart_management", "openshift_container_platform"}
)

// SubModel is the struct for the json request/response from subscriptions API
type SubModel struct {
	Name  string `json:"name"`
	Rules Rules `json:"rules"`
}

type Rules struct {
	MatchProducts []MatchProducts `json:"matchProducts,omitempty"`
	ExcludeProducts []ExcludeProducts `json:"excludeProducts,omitempty"`
}

type MatchProducts struct {
	SkuCodes []string `json:"skuCodes,omitempty"`
}

type ExcludeProducts struct {
	SkuCodes []string `json:"skuCodes,omitempty"`
}

type YAMLSkus []struct {
	Name string `yaml:"name"`
	Skus map[string]map[string]bool `yaml:"skus,omitempty"`
	AccNum bool `yaml:"use_valid_acc_num,omitempty"`
}


// assertEq compares two slices of strings and returns true if they are equalzs
func assertEq(test []string, ans []string) bool {
    return reflect.DeepEqual(test, ans)
}

// getClient sets up the http client for the subscriptions API
func getClient(cfg *cfg.EntitlementsConfig) *http.Client {

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

func getCurrent(client *http.Client, url string) (SubModel, error) {
	resp, err := client.Get(url)
	if err != nil {
		log.Fatal(err)
		return SubModel{}, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return SubModel{}, err
	}

	var currentSubs SubModel
	json.Unmarshal(data, &currentSubs)
	return currentSubs, nil
}

func getUpdates(cfg *viper.Viper) (YAMLSkus, error){
	var env string = strings.Split(cfg.GetString("SUBS_HOST"), ".")[1]

	if env == "api" {
		env = "prod"
	}
	if env == "dev" {
		env = "ci"
	}
	resp, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/RedHatInsights/entitlements-config/master/configs/%s/bundles.yml", env))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal()
		return nil, err
	}
	
	var m YAMLSkus
	yaml.Unmarshal(data, &m)

	return m, nil

}

func postUpdates(cfg *viper.Viper, client *http.Client, data []byte) error {
	url := fmt.Sprintf("%s%s", cfg.GetString("SUBS_HOST"), cfg.GetString("SUB_API_BASE_PATH"))
	req, err := client.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer req.Body.Close()

	return nil

}


func main() {
	c := cfg.GetConfig()
	client := getClient(c)
	options := c.Options
	for _, endpoint := range endpoints {
		skus := make(map[string][]string)
		current_skus := make(map[string][]string)
		url := fmt.Sprintf("%s%s",
						   options.GetString("SUBS_HOST"),
						   options.GetString("SUB_API_BASE_PATH"))
		current, err := getCurrent(client, url + endpoint)
		if err != nil {
			os.Exit(1)
		}

		sku_updates, err := getUpdates(options)
		if err != nil {
			os.Exit(1)
		}
		for _, v := range sku_updates {
			if v.Name == endpoint {
				for sku := range v.Skus {
					skus[endpoint] = append(skus[endpoint], sku)
				}
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
			var m []MatchProducts
			m = append(m, MatchProducts{SkuCodes: skus[endpoint]})
			v := SubModel{
				Name: endpoint,
				Rules: Rules{
					MatchProducts: m,
				},
			}
			b, err := json.Marshal(v)
			if err != nil {
				log.Fatal(err)
			}
			err = postUpdates(options, client, b)
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			} else {
				fmt.Printf("Updated %s\n", endpoint)
			}
		}
	}
}
