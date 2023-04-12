package bop

import (
	"bytes"
	"encoding/json"
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
	Users []string
}

type responseBody struct {
	Records []UserDetail
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

func (c *Client) GetUser(userName string) (*UserDetail, error) {
	buf, err := makeRequestBody(userName)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.url, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-rh-clientid", c.clientId)
	req.Header.Set("x-rh-apitoken", c.token)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	bopRequestTime.Observe(time.Since(start).Seconds())

	if err != nil {
		return nil, err
	}
	var decoded responseBody
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return &decoded.Records[0], nil
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

func GetClient(debug bool) (Bop, error) {
	if debug {
		return &Mock{
			OrgId: DEFAULT_ORG_ID,
		}, nil
	}
	return &Client{
		clientId:   config.GetConfig().Options.GetString(config.Keys.BOPClientID),
		token:      config.GetConfig().Options.GetString(config.Keys.BOPToken),
		url:        config.GetConfig().Options.GetString(config.Keys.BOPURL),
		httpClient: http.Client{},
	}, nil
}
