package ams

import (
	"fmt"

	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type Mock struct{}

var _ AMSInterface = &Mock{}

var MockGetQuotaCost = func(organizationId string) (*v1.QuotaCost, error) {
    quotaCost, err := v1.NewQuotaCost().QuotaID("seat|ansible.wisdom").Build()
    if err != nil {
        return nil, err
    }
    return quotaCost, nil
}
func (c *Mock) GetQuotaCost(organizationId string) (*v1.QuotaCost, error) {
    return MockGetQuotaCost(organizationId)
}

var MockGetSubscription = func(subscriptionId string) (*v1.Subscription, error) {
    if subscriptionId == "" {
        return nil, fmt.Errorf("subscriptionId cannot be an empty string")
    }
    subscription, err := v1.NewSubscription().
        ID(subscriptionId).
        OrganizationID("AMSORG4384938490324").
        Build()
    if err != nil {
        return nil, err
    }
    return subscription, nil
}
func (c *Mock) GetSubscription(subscriptionId string) (*v1.Subscription, error) {
    return MockGetSubscription(subscriptionId)
}

var MockDeleteSubscription = func(subscriptionId string) error {
    return nil
}
func (c *Mock) DeleteSubscription(subscriptionId string) error {
    return MockDeleteSubscription(subscriptionId)
}

var MockQuotaAuthorization = func(accountUsername, quotaVersion string) (*v1.QuotaAuthorizationResponse, error) {
    resp, err := v1.NewQuotaAuthorizationResponse().Allowed(true).Build()
    return resp, err
}
func (c *Mock) QuotaAuthorization(accountUsername, quotaVersion string) (*v1.QuotaAuthorizationResponse, error) {
    return MockQuotaAuthorization(accountUsername, quotaVersion)
}

var MockGetSubscriptions = func(organizationId string, statuses []string, size, page int) (*v1.SubscriptionList, error) {
    lst, err := v1.NewSubscriptionList().
        Items(
            v1.NewSubscription().
                Creator(v1.NewAccount().Username("testuser").FirstName("test").LastName("user")).
                Plan(v1.NewPlan().Type("AnsibleWisdom").Name("AnsibleWisdom")).
                Status("Active"),
        ).Build()
    if err != nil {
        return nil, err
    }
    return lst, nil
}
func (c *Mock) GetSubscriptions(organizationId string, statuses []string, size, page int) (*v1.SubscriptionList, error) {
    return MockGetSubscriptions(organizationId, statuses, size, page)
}

var MockConvertUserOrgId = func(userOrgId string) (string, error) {
    if userOrgId == "4384938490324" {
        return "AMSORG4384938490324", nil
    }
    return "AMSORG1", nil
}
func (c *Mock) ConvertUserOrgId(userOrgId string) (string, error) {
    return MockConvertUserOrgId(userOrgId)
}
