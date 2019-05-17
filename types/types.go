package types

type EntitlementsSection struct {
	IsEntitled bool `json:"is_entitled"`
}

type SubscriptionsResponse struct {
	StatusCode int
	Data []string
	CacheHit bool
}

type EntitlementsResponse struct {
	HybridCloud    EntitlementsSection `json:"hybrid_cloud"`
	Insights       EntitlementsSection `json:"insights"`
	Openshift      EntitlementsSection `json:"openshift"`
	SmartMangement EntitlementsSection `json:"smart_management"`
}
