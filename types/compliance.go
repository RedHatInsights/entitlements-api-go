package types

type User struct {
	Id    string `json:"id,omitempty"`
	Login string `json:"login,omitempty"`
}

type Account struct {
	Primary bool `json:"primary"`
}

type ComplianceScreeningRequest struct {
	User    User    `json:"user"`
	Account Account `json:"account"`
}

type ComplianceScreeningResponse struct {
	Result      string `json:"result"`
	Description string `json:"description"`
}
