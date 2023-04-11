package bop

// TODO: Factor this out
const DEFAULT_ORG_ID string = "4384938490324"

type UserDetail struct {
	UserName string
	OrgId    string
}

type Bop interface {
	GetUser(userName string) (*UserDetail, error)
}

type Client struct {
}

var _ Bop = &Client{}

func (c *Client) GetUser(userName string) (*UserDetail, error) {
	return nil, nil
}

type Mock struct {
	OrgId string
}

var _ Bop = &Mock{}

func (m *Mock) GetUser(userName string) (*UserDetail, error) {
	return &UserDetail{
		UserName: userName,
		OrgId:    m.OrgId,
	}, nil
}

func GetClient(debug bool) (Bop, error) {
	if debug {
		return &Mock{
			OrgId: DEFAULT_ORG_ID,
		}, nil
	}
	return &Client{}, nil
}
