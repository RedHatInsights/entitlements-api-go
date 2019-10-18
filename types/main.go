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

// Bundle is a struct that is used to unmarshal the bundle info from bundles.yml
type Bundle struct {
	Name           string   `yaml:"name"`
	UseValidAccNum bool     `yaml:"use_valid_acc_num"`
	Skus           []string `yaml:"skus"`
}

type DependencyErrorDetails struct {
	DependencyFailure bool   `json:"dependency_failure"`
	Service 					string `json:"service"`
	Status 						int    `json:"status"`
	Endpoint					string `json:"endpoint"`
	Message 					string `json:"message"`
}

type DependencyErrorResponse struct {
	Error DependencyErrorDetails `json:error`
}
