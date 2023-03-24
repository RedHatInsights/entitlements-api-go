package types

type Seat struct {
	SubscriptionId  string `json:"subscription_id"`
	AccountUsername string `json:"account_username"`
}

type SeatList struct {
	Seats []Seat ``
}
