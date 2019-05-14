package types

type Internal struct {
	Org_id string `json:org_id`
}

type XRhIdentity struct {
	Account_number string `json:"account_number"`
	Internal Internal `json:"internal"`
}

type EntitlementsSection struct {
	Is_entitled bool `json:"is_entitled"`
}

type EntitlementsResponse struct {
	Hybrid_cloud    EntitlementsSection `json:"hybrid_cloud"`
	Insights        EntitlementsSection `json:"insights"`
	Openshift       EntitlementsSection `json:"openshift"`
	Smart_mangement EntitlementsSection `json:"smart_management"`
}
