package types

// EntitlementsSection is a struct representing { "is_entitled": bool, "is_trial": bool } on the SubscriptionsResponse
type EntitlementsSection struct {
	IsEntitled bool `json:"is_entitled"`
	IsTrial    bool `json:"is_trial"`
}

// FeatureResponse is a struct that is used to unmarshal the data that comes back from the
// Feature Service
type FeatureResponse struct {
	StatusCode int
	Body       string
	Error      error
	Data       FeatureStatus
	CacheHit   bool
	Url        string
}

type Feature struct {
	Name     string `json:"name"`
	IsEval   bool   `json:"isEval"`
	IsEntitled bool   `json:"isEntitled"`
}

type FeatureStatus struct {
	Features []Feature `json:"features"`
}

// Bundle is a struct that is used to unmarshal the bundle info from bundles.yml
type Bundle struct {
	Name           string   `yaml:"name"`
	UseValidAccNum bool     `yaml:"use_valid_acc_num"`
	UseValidOrgId  bool     `yaml:"use_valid_org_id"`
	UseIsInternal  bool     `yaml:"use_is_internal"`
	Skus           []string `yaml:"skus"`
}

// DependencyErrorDetails is a struct that is used to marshal failure details
// from failed requests to external services
type DependencyErrorDetails struct {
	DependencyFailure bool   `json:"dependency_failure"`
	Service           string `json:"service"`
	Status            int    `json:"status"`
	Endpoint          string `json:"endpoint"`
	Message           string `json:"message"`
}

// DependencyErrorResponse is a struct that is used to marshal an error response
// based on details from a failed request to external services
type DependencyErrorResponse struct {
	Error DependencyErrorDetails `json:"error"`
}

// RequestErrorDetails is used to describe what was invalid about a bad request to entitlements
type RequestErrorDetails struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// RequestErrorResponse is used to marshal an error response based on details from a bad request to entitlements
type RequestErrorResponse struct {
	Error RequestErrorDetails `json:"error"`
}

// SubModel is the struct for GET and POST data for subscriptions
type SubModel struct {
	Name  string `json:"name"`
	Rules []Rules  `json:"rules"`
}

// Rules contains match and exclude product arrays
type Rules struct {
	MatchProducts   []MatchProducts   `json:"matchProducts,omitempty"`
	ExcludeProducts []ExcludeProducts `json:"excludeProducts,omitempty"`
}

// MatchProducts contains the SkuCodes array
type MatchProducts struct {
	SkuCodes []string `json:"skuCodes,omitempty"`
}

// ExcludeProducts contains the SkuCodes array
type ExcludeProducts struct {
	SkuCodes []string `json:"skuCodes,omitempty"`
}
