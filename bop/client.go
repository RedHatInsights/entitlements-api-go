package bop

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var bopRequestTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "bop_service_request_time_taken",
	Help:    "bop service latency distributions",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})
var bopFailure = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "back_office_proxy_service_failure",
		Help: "Total number of Back Office Proxy service failures. A failure means a request to bop returned a non 2xx.",
	},
	[]string{"code"},
)

type UserDetail struct {
	UserName string `json:"username"`
	OrgId    string `json:"org_id"`
}

type UserDetailError struct {
	Message    string `json:"message"`
	StatusCode int
	UserName   string
}

func (e *UserDetailError) Error() string {
	return fmt.Sprintf("BOP GetUser error [%s], http status code [%d], username [%s]", e.Message, e.StatusCode, e.UserName)
}

type Bop interface {
	GetUser(userName string) (*UserDetail, error)
}

type Client struct {
	clientId   string
	token      string
	url        string
	httpClient http.Client
	env        string
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
	req.Header.Set("x-rh-insights-env", c.env)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	bopRequestTime.Observe(time.Since(start).Seconds())

	if err != nil {
		return nil, fmt.Errorf("Error from trying to send BOP GetUser request [%w]", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		incBopFailure(resp.StatusCode)

		var decodedError UserDetailError
		if err = json.NewDecoder(resp.Body).Decode(&decodedError); err != nil {
			return nil, fmt.Errorf("Unable to decode BOP GetUser response [%w], request status [%d]", err, resp.StatusCode)
		}

		decodedError.StatusCode = resp.StatusCode
		decodedError.UserName = userName
		return nil, &decodedError
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		incBopFailure(resp.StatusCode)
		return nil, fmt.Errorf("Internal server error received from BOP GetUser request, status [%d]", resp.StatusCode)
	}

	var decoded []UserDetail
	if err = json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		incBopFailure(resp.StatusCode)
		return nil, fmt.Errorf("Unable to decode BOP GetUser response [%w], request status [%d]", err, resp.StatusCode)
	}

	if len(decoded) < 1 {
		incBopFailure(http.StatusNotFound)
		return nil, &UserDetailError{
			Message:    "No users found for given username",
			StatusCode: http.StatusNotFound,
			UserName:   userName,
		}
	}

	return &decoded[0], nil
}

func incBopFailure(statusCode int) {
	bopFailure.WithLabelValues(strconv.Itoa(statusCode)).Inc()
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
	options := config.GetConfig().Options

	if debug {
		return &Mock{
			OrgId: options.GetString(config.Keys.BOPMockOrgId),
		}, nil
	}

	clientId := options.GetString(config.Keys.BOPClientID)
	token := options.GetString(config.Keys.BOPToken)
	url := options.GetString(config.Keys.BOPURL)
	env := options.GetString(config.Keys.BOPEnv)

	if err := validateBOPSettings(clientId, token, url); err != nil {
		return nil, err
	}

	return &Client{
		clientId: clientId,
		token:    token,
		url:      url,
		env:      env,
		httpClient: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: config.GetConfig().RootCAs,
				},
			},
		},
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
