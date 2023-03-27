package ams

import (
	"context"

	"github.com/RedHatInsights/entitlements-api-go/config"
	sdk "github.com/openshift-online/ocm-sdk-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type AMSInterface interface {
	GetQuotaCost(organizationId string) (*v1.QuotaCost, error)
	GetSubscription(subscriptionId string) (*v1.Subscription, error)
	GetSubscriptions() (*v1.SubscriptionList, error)
	DeleteSubscription(subscriptionId string) error
	QuotaAuthorization(accountUsername string)
}

var _ AMSInterface = &TestClient{}

type TestClient struct{}

func (c *TestClient) GetQuotaCost(organizationId string) (*v1.QuotaCost, error) {
	quotaCost, err := v1.NewQuotaCost().QuotaID("seat|ansible.wisdom").Build()
	if err != nil {
		return nil, err
	}
	return quotaCost, nil
}

func (c *TestClient) GetSubscription(subscriptionId string) (*v1.Subscription, error) {
	subscription, err := v1.NewSubscription().Build()
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *TestClient) DeleteSubscription(subscriptionId string) error {
	return nil
}

// TODO: waiting on updates to the ocm sdk
func (c *TestClient) QuotaAuthorization(accountUsername string) {}

func (c *TestClient) GetSubscriptions() (*v1.SubscriptionList, error) {
	lst, err := v1.NewSubscriptionList().Items(
		v1.NewSubscription().Creator(
			v1.NewAccount().Username("testuser"),
		),
	).Build()
	if err != nil {
		return nil, err
	}
	return lst, nil
}

var _ AMSInterface = &Client{}

type Client struct {
	client *sdk.Connection
}

func NewClient() (*Client, error) {

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
	resp, err := c.client.AccountsMgmt().V1().Organizations().Organization(organizationId).QuotaCost().List().Search(
		"quota_id LIKE 'seat|ansible.wisdom%'",
	).Send()
	if err != nil {
		return nil, err
	}
	return resp.Items().Get(0), nil
}

func (c *Client) GetSubscription(subscriptionId string) (*v1.Subscription, error) {
	subscription, err := v1.NewSubscription().Build()
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *Client) GetSubscriptions() (*v1.SubscriptionList, error) {
	resp, err := c.client.AccountsMgmt().V1().Subscriptions().List().Search(
		"quota_id LIKE 'seat|ansible.wisdom'%").Send()
	if err != nil {
		return nil, err
	}
	return resp.Items(), nil
}

func (c *Client) DeleteSubscription(subscriptionId string) error {
	return nil
}

// TODO: waiting on updates to the ocm sdk
func (c *Client) QuotaAuthorization(accountUsername string) {}
