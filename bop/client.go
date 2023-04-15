package bop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TODO: Factor this out
const DEFAULT_ORG_ID string = "4384938490324"

var bopRequestTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "bop_service_request_time_taken",
	Help:    "bop service latency distributions",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})

type UserDetail struct {
	UserName string `json:"username"`
	OrgId    string `json:"org_id"`
}

type Bop interface {
	GetUser(userName string) (*UserDetail, error)
}

type Client struct {
	clientId   string
	token      string
	url        string
	httpClient http.Client
}

var _ Bop = &Client{}

type userRequest struct {
	Users []string `json:"users"`
}

func makeRequestBody(userName string) (*bytes.Buffer, error) {
	requestBody := userRequest{
		Users: []string{userName},
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(encoded), nil
}

func makeRequest(userName, url string) (*http.Request, error) {
	buf, err := makeRequestBody(userName)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *Client) GetUser(userName string) (*UserDetail, error) {
	req, err := makeRequest(userName, c.url)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-rh-clientid", c.clientId)
	req.Header.Set("x-rh-apitoken", c.token)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	bopRequestTime.Observe(time.Since(start).Seconds())

	if err != nil {
		return nil, fmt.Errorf("Error from sending BOP GetUser request [%w]", err)
	}
	var decoded []UserDetail
	if err = json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("Error from decoding BOP GetUser response [%w]", err)
	}

	if len(decoded) < 1 {
		return nil, fmt.Errorf("no records returned when searching for '%s'", userName)
	}

	return &decoded[0], nil
}

type Mock struct {
	OrgId string
}

var _ Bop = &Mock{}

func (m *Mock) GetUser(userName string) (*UserDetail, error) {
	return &UserDetail{
		UserName: userName,
		OrgId:    m.OrgId,
	}, nil
}

func NewClient(debug bool) (Bop, error) {
	if debug {
		return &Mock{
			OrgId: DEFAULT_ORG_ID,
		}, nil
	}

	options := config.GetConfig().Options
	clientId := options.GetString(config.Keys.BOPClientID)
	token := options.GetString(config.Keys.BOPToken)
	url := options.GetString(config.Keys.BOPURL)

	if err := validateBOPSettings(clientId, token, url); err != nil {
		return nil, err
	}

	return &Client{
		clientId:   clientId,
		token:      token,
		url:        url,
		httpClient: http.Client{},
	}, nil
}

func validateBOPSettings(clientId string, token string, url string) error {
	missingConfig := make([]string, 0)

	if clientId == "" {
		missingConfig = append(missingConfig, config.Keys.BOPClientID)
	}

	if token == "" {
		missingConfig = append(missingConfig, config.Keys.BOPToken)
	}

	if url == "" {
		missingConfig = append(missingConfig, config.Keys.BOPURL)
	}

	if len(missingConfig) > 0 {
		return fmt.Errorf("Error configuring BOP client. Must provide the following env variables which are missing: %v", missingConfig)
	}

	return nil
}
