package types

// EntitlementsSection is a struct representing { "is_entitled": bool } on the SubscriptionsResponse
type EntitlementsSection struct {
	IsEntitled bool `json:"is_entitled"`
}

// SubscriptionsResponse is a struct that is used to unmarshal the data that comes back from the
// Suscriptions Service
type SubscriptionsResponse struct {
	StatusCode int
	Body       string
	Error      error
	Data       []string
	CacheHit   bool
}

// EntitlementsResponse is the struct that is used to marshal/unmarshal the response from Entitlemens API
type EntitlementsResponse struct {
	HybridCloud    EntitlementsSection `json:"hybrid_cloud"`
	Insights       EntitlementsSection `json:"insights"`
	Openshift      EntitlementsSection `json:"openshift"`
	SmartMangement EntitlementsSection `json:"smart_management"`
}
