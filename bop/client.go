package bop

type UserDetail struct {
}

type Bop interface {
	GetUser(userName string) (*UserDetail, error)
}

type Client struct {
}

var _ Bop = &Client{}

func (*Client) GetUser(userName string) (*UserDetail, error) {
	return nil, nil
}

type Mock struct{}

var _ Bop = &Mock{}

func (*Mock) GetUser(userName string) (*UserDetail, error) {
	return nil, nil
}
