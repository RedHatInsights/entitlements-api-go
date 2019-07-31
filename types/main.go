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

// SubscriptionProducts is a struct that is used to unmarshal the entries field that comes back from the
// response of the Subscription Service
type SubscriptionProducts struct {
	Sku string `json:"sku"`
}

// SubscriptionBody is a struct that is used to unmarshal the data that comes back in the Body
// of the response of Subscriptions Service
type SubscriptionBody struct {
	SubscriptionProducts []SubscriptionProducts
}

// EntitlementsResponse is the struct that is used to marshal/unmarshal the response from Entitlemens API
type EntitlementsResponse struct {
	HybridCloud    EntitlementsSection `json:"hybrid_cloud"`
	Insights       EntitlementsSection `json:"insights"`
	Openshift      EntitlementsSection `json:"openshift"`
	SmartMangement EntitlementsSection `json:"smart_management"`
}
