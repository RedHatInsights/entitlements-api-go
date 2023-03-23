package ocm

import (
	"context"
	"fmt"

	"github.com/redhatinsights/mbop/internal/config"
	"github.com/redhatinsights/mbop/internal/models"
)

type OCM interface {
	InitSdkConnection(ctx context.Context) error
	CloseSdkConnection()
	GetUsers(users models.UserBody, q models.UserV1Query) (models.Users, error)
	GetAccountV3Users(orgID string, q models.UserV3Query) (models.Users, error)
	GetAccountV3UsersBy(orgID string, q models.UserV3Query, body models.UsersByBody) (models.Users, error)
	GetOrgAdmin([]models.User) (models.OrgAdminResponse, error)
}

// re-declaring ams constant here to avoid circular module importing
const amsModule = "ams"
const mockModule = "mock"

func NewOcmClient() (OCM, error) {
	var client OCM

	switch config.Get().UsersModule {
	case amsModule:
		client = &SDK{}
	case mockModule:
		client = &SDKMock{}
	default:
		return nil, fmt.Errorf("unsupported users module %q", config.Get().UsersModule)
	}

	return client, nil
}
