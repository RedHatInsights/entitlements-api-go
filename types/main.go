package types

// EntitlementsSection is a struct representing { "is_entitled": bool } on the SubscriptionsResponse
type EntitlementsSection struct {
	IsEntitled bool `json:"is_entitled"`
}

// SubscriptionsResponse is a struct that is used to unmarshal the data that comes back from the
// Subscriptions Service
type SubscriptionsResponse struct {
	StatusCode int
	Body       string
	Error      error
	Data       []string
	CacheHit   bool
}

// Entries is a struct that is used to unmarshal the entries field that comes back from the
// response of the Subscription Service
type Entries struct {
	Value string
}

// SubscriptionDetails is a struct that is used to unmarshal the data that comes back in the Body
// of the response of Subscriptions Service
type SubscriptionDetails struct {
	Entries []Entries
}

// EntitlementsResponse is the struct that is used to marshal/unmarshal the response from Entitlemens API
type EntitlementsResponse struct {
	HybridCloud     EntitlementsSection `json:"hybrid_cloud"`
	Insights        EntitlementsSection `json:"insights"`
	Openshift       EntitlementsSection `json:"openshift"`
	SmartManagement EntitlementsSection `json:"smart_management"`
	Ansible         EntitlementsSection `json:"ansible"`
	Migrations      EntitlementsSection `json:"migrations"`
}
