package ams

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/api"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/karlseguin/ccache"
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
var orgListTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "org_list_service_request_time_taken",
	Help:    "org_list service latency distributions.",
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
	GetSubscriptions(organizationId string, searchParams api.GetSeatsParams, size, page int) (*v1.SubscriptionList, error)
	DeleteSubscription(subscriptionId string) error
	QuotaAuthorization(accountUsername, quotaVersion string) (*v1.QuotaAuthorizationResponse, error)
	ConvertUserOrgId(userOrgId string) (string, error)
}

type ClientError struct {
	Message    string
	StatusCode int
	OrgId      string
	AmsOrgId   string
}

func (e *ClientError) Error() string {
	b := strings.Builder{}

	b.WriteString(e.Message)
	if e.OrgId != "" {
		b.WriteString(fmt.Sprintf(" [OrgId: %s]", e.OrgId))
	}
	if e.AmsOrgId != "" {
		b.WriteString(fmt.Sprintf(" [AMS OrgId: %s]", e.AmsOrgId))
	}

	return b.String()
}

type Client struct {
	client *sdk.Connection
	cache  ccache.Cache
}

var _ AMSInterface = &Client{}

func NewClient(debug bool) (AMSInterface, error) {

	if debug {
		return &Mock{}, nil
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
		cache:  *ccache.New(ccache.Configure()),
	}, err
}

func (c *Client) GetQuotaCost(organizationId string) (*v1.QuotaCost, error) {

	amsOrgId, err := c.ConvertUserOrgId(organizationId)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := c.client.AccountsMgmt().V1().Organizations().Organization(amsOrgId).QuotaCost().List().Search(
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

func (c *Client) GetSubscriptions(organizationId string, searchParams api.GetSeatsParams, size, page int) (*v1.SubscriptionList, error) {
	amsOrgId, err := c.ConvertUserOrgId(organizationId)
	if err != nil {
		return nil, err
	}

	queryBuilder := NewQueryBuilder().
		Like("plan.id", "AnsibleWisdom").
		And().
		Equals("organization_id", amsOrgId)

	if statuses, err := buildStatusSearch(searchParams.Status); statuses != nil && err == nil {
		queryBuilder = queryBuilder.And().In("status", statuses)
	} else if statuses == nil && err != nil {
		return nil, &ClientError{
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			OrgId:      organizationId,
			AmsOrgId:   amsOrgId,
		}
	}

	if isSearchStrValid(searchParams.AccountUsername) {
		queryBuilder = queryBuilder.And().Equals("creator.username", *searchParams.AccountUsername)
	}

	if isSearchStrValid(searchParams.Email) {
		queryBuilder = queryBuilder.And().Equals("creator.email", *searchParams.Email)
	}

	if isSearchStrValid(searchParams.FirstName) {
		queryBuilder = queryBuilder.And().Equals("creator.first_name", *searchParams.FirstName)
	}

	if isSearchStrValid(searchParams.LastName) {
		queryBuilder = queryBuilder.And().Equals("creator.last_name", *searchParams.LastName)
	}

	query := queryBuilder.Build()

	start := time.Now()
	req := c.client.AccountsMgmt().V1().Subscriptions().List().
		Search(query).
		Parameter("fetchAccounts", true).
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

func (c *Client) QuotaAuthorization(accountUsername, quotaVersion string) (*v1.QuotaAuthorizationResponse, error) {

	rr := v1.NewReservedResource().
		ResourceName("ansible.wisdom").
		ResourceType("seat").
		Count(1).
		BYOC(false)

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
	postResponse, err := c.client.AccountsMgmt().V1().QuotaAuthorizations().Post().Request(req).Send()
	return postResponse.Response(), err
}

// ConvertUserOrgId Convert a user org id from rh-identity header to an ams org id
func (c *Client) ConvertUserOrgId(userOrgId string) (string, error) {
	item := c.cache.Get(userOrgId)
	if item != nil && !item.Expired() {
		converted := item.Value().(string)
		l.Log.WithFields(logrus.Fields{"ams_org_id": converted, "org_id": userOrgId}).Debug("found converted ams org id in cache")
		return converted, nil
	}

	if valid, _ := validateOrgIdPattern(userOrgId); !valid {
		return "", &ClientError{
			Message:    "invalid user org id - id contains non alpha numeric characters",
			StatusCode: http.StatusInternalServerError,
			OrgId:      userOrgId,
			AmsOrgId:   "",
		}
	}

	start := time.Now()
	listResp, err := c.client.
		AccountsMgmt().V1().
		Organizations().List().
		Search(NewQueryBuilder().Equals("external_id", userOrgId).Build()).
		Send()

	orgListTime.Observe(time.Since(start).Seconds())
	if err != nil {
		return "", err
	}

	converted, err := listResp.Items().Get(0).ID(), nil

	if converted == "" {
		return "", &ClientError{
			Message:    "no corresponding ams org id found for user org",
			StatusCode: http.StatusBadRequest,
			OrgId:      userOrgId,
			AmsOrgId:   "",
		}
	}

	if valid, _ := validateOrgIdPattern(converted); !valid {
		return "", &ClientError{
			Message:    "invalid ams org id - id contains non alpha numeric characters",
			StatusCode: http.StatusInternalServerError,
			OrgId:      userOrgId,
			AmsOrgId:   converted,
		}
	}

	c.cache.Set(userOrgId, converted, time.Minute*30)

	l.Log.WithFields(logrus.Fields{"ams_org_id": converted, "org_id": userOrgId}).Debug("converted org id to ams org ig")

	return converted, err
}

func buildStatusSearch(statuses *api.Status) (api.Status, error) {
	if statuses == nil || len(*statuses) == 0 {
		return nil, nil
	}

	// ams is case sensitive when searching on status, so title case all input for the consumer
	caser := cases.Title(language.English)
	titleCased := api.Status{}
	for _, status := range *statuses {
		statusType := api.GetSeatsParamsStatus(caser.String(status))
		switch statusType {
		case api.Active, api.Deprovisioned:
			titleCased = append(titleCased, string(statusType))
		case "":
		default:
			return nil, fmt.Errorf("provided status '%s' is an unsupported status to query seats for, check apispec for list of supported statuses", status)
		}
	}

	return titleCased, nil
}

func isSearchStrValid(val *string) bool {
	return val != nil && *val != "" && strings.TrimSpace(*val) != ""
}

func validateOrgIdPattern(orgId string) (match bool, err error) {
	return regexp.MatchString("^[a-zA-Z0-9]+$", orgId)
}
