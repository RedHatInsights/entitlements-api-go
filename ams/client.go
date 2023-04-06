package ams

import (
	"context"
	"fmt"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	sdk "github.com/openshift-online/ocm-sdk-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var quotaCostTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "quota_cost_service_request_time_taken",
	Help:    "quota_cost service latency distributions.",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})
var getSubscriptionTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "get_subscription_service_request_time_taken",
	Help:    "get_subscription service latency distributions.",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})
var getSubscriptionsTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "get_subscriptions_service_request_time_taken",
	Help:    "get_subscriptions service latency distributions.",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})
var deleteSubscriptionTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "delete_subscription_service_request_time_taken",
	Help:    "delete_subscription service latency distributions.",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})
var quotaAuthorizationTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "quota_authorization_service_request_time_taken",
	Help:    "quota_authorization service latency distributions.",
	Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})

type AMSInterface interface {
	GetQuotaCost(organizationId string) (*v1.QuotaCost, error)
	GetSubscription(subscriptionId string) (*v1.Subscription, error)
	GetSubscriptions(size, page int) (*v1.SubscriptionList, error)
	DeleteSubscription(subscriptionId string) error
	QuotaAuthorization(accountUsername, quotaVersion string) (*v1.QuotaAuthorizationsPostResponse, error)
}

type TestClient struct{}

var _ AMSInterface = &TestClient{}

func (c *TestClient) GetQuotaCost(organizationId string) (*v1.QuotaCost, error) {
	quotaCost, err := v1.NewQuotaCost().QuotaID("seat|ansible.wisdom").Build()
	if err != nil {
		return nil, err
	}
	return quotaCost, nil
}

func (c *TestClient) GetSubscription(subscriptionId string) (*v1.Subscription, error) {
	if subscriptionId == "" {
		return nil, fmt.Errorf("subscriptionId cannot be an empty string")
	}
	subscription, err := v1.NewSubscription().
		ID(subscriptionId).
		OrganizationID("4384938490324").
		Build()
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *TestClient) DeleteSubscription(subscriptionId string) error {
	return nil
}

// TODO: waiting on updates to the ocm sdk
func (c *TestClient) QuotaAuthorization(accountUsername, quotaVersion string) (*v1.QuotaAuthorizationsPostResponse, error) {
	return nil, nil
}

func (c *TestClient) GetSubscriptions(size, page int) (*v1.SubscriptionList, error) {
	lst, err := v1.NewSubscriptionList().
		Items(
			v1.NewSubscription().
				Creator(v1.NewAccount().Username("testuser")).
				Plan(v1.NewPlan().Type("AnsibleWisdom").Name("AnsibleWisdom")),
		).Build()
	if err != nil {
		return nil, err
	}
	return lst, nil
}

type Client struct {
	client *sdk.Connection
}

var _ AMSInterface = &Client{}

func NewClient(debug bool) (AMSInterface, error) {

	if debug {
		return &TestClient{}, nil
	}

	logger, err := logging.NewGoLoggerBuilder().Debug(false).Build()
	if err != nil {
		return nil, err
	}

	cfg := config.GetConfig()

	clientId := cfg.Options.GetString(config.Keys.ClientID)
	secret := cfg.Options.GetString(config.Keys.ClientSecret)
	tokenUrl := cfg.Options.GetString(config.Keys.TokenURL)
	amsUrl := cfg.Options.GetString(config.Keys.AMSHost)

	client, err := sdk.NewConnectionBuilder().
		Logger(logger).
		Client(clientId, secret).
		TokenURL(tokenUrl).
		URL(amsUrl).
		BuildContext(context.Background())

	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
	}, err
}

func (c *Client) GetQuotaCost(organizationId string) (*v1.QuotaCost, error) {
	start := time.Now()
	resp, err := c.client.AccountsMgmt().V1().Organizations().Organization(organizationId).QuotaCost().List().Search(
		"quota_id LIKE 'seat|ansible.wisdom%'",
	).Send()
	quotaCostTime.Observe(time.Since(start).Seconds())
	if err != nil {
		return nil, err
	}
	return resp.Items().Get(0), nil
}

func (c *Client) GetSubscription(subscriptionId string) (*v1.Subscription, error) {
	start := time.Now()
	resp, err := c.client.AccountsMgmt().V1().Subscriptions().Subscription(subscriptionId).Get().Send()
	getSubscriptionTime.Observe(time.Since(start).Seconds())
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c *Client) GetSubscriptions(size, page int) (*v1.SubscriptionList, error) {
	start := time.Now()
	req := c.client.AccountsMgmt().V1().Subscriptions().List().
		Search("quota_id LIKE 'seat|ansible.wisdom'%").
		Size(size).
		Page(page)

	resp, err := req.Send()
	getSubscriptionsTime.Observe(time.Since(start).Seconds())
	if err != nil {
		return nil, err
	}
	return resp.Items(), nil
}

func (c *Client) DeleteSubscription(subscriptionId string) error {
	start := time.Now()
	_, err := c.client.AccountsMgmt().V1().Subscriptions().Subscription(subscriptionId).Delete().Send()
	deleteSubscriptionTime.Observe(time.Since(start).Seconds())
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) QuotaAuthorization(accountUsername, quotaVersion string) (*v1.QuotaAuthorizationsPostResponse, error) {

	rr := v1.NewReservedResource().
		ResourceName("ansible.wisdom").
		ResourceType("seat")

	req, err := v1.NewQuotaAuthorizationRequest().
		AccountUsername(accountUsername).
		Reserve(true).
		ProductID("AnsibleWisdom").
		Resources(rr).
		QuotaVersion(quotaVersion).
		Build()

	if err != nil {
		return nil, err
	}
	start := time.Now()
	defer quotaAuthorizationTime.Observe(time.Since(start).Seconds())
	return c.client.AccountsMgmt().V1().QuotaAuthorizations().Post().Request(req).Send()
}
