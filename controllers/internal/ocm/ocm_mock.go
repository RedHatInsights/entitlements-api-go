package ocm

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"

	"github.com/google/uuid"
	"github.com/redhatinsights/mbop/internal/models"
)

type SDKMock struct{}

func (ocm *SDKMock) InitSdkConnection(_ context.Context) error {
	return nil
}

func (ocm *SDKMock) GetUsers(u models.UserBody, _ models.UserV1Query) (models.Users, error) {
	var users models.Users

	if u.Users == nil {
		return users, nil
	}

	if u.Users[0] == "errorTest" {
		return users, fmt.Errorf("internal AMS Error")
	}

	for _, user := range u.Users {
		displayNameNum, err := rand.Int(rand.Reader, big.NewInt(99-0))
		if err != nil {
			return users, err
		}

		users.AddUser(models.User{
			Username:      user,
			ID:            uuid.New().String(),
			Email:         "lub@dub.com",
			FirstName:     "test",
			LastName:      "case",
			AddressString: "https://usersTest.com",
			IsActive:      true,
			IsInternal:    false,
			Locale:        "en_US",
			OrgID:         user,
			DisplayName:   "FedRAMP" + strconv.Itoa(int(displayNameNum.Int64())),
			Type:          "User",
		})
	}

	return users, nil
}

func (ocm *SDKMock) GetOrgAdmin(users []models.User) (models.OrgAdminResponse, error) {
	response := models.OrgAdminResponse{}

	if users[0].ID == "23456" {
		return response, nil
	}

	if users[0].ID == "errorTest" {
		return response, fmt.Errorf("error retrieving Role Bindings")
	}

	for _, user := range users {
		response[user.ID] = models.OrgAdmin{
			ID:         user.ID,
			IsOrgAdmin: true,
		}
	}

	return response, nil
}

func (ocm *SDKMock) GetAccountV3Users(orgID string, q models.UserV3Query) (models.Users, error) {
	users := models.Users{Users: []models.User{}}

	if orgID == "empty" {
		return users, nil
	}

	if orgID == "errorTest" {
		return users, fmt.Errorf("error retrieving V3 Users")
	}

	for i := q.Offset; i < q.Limit; i++ {
		displayNameNum, err := rand.Int(rand.Reader, big.NewInt(99-0))
		if err != nil {
			return users, err
		}

		users.AddUser(models.User{
			Username:      "TestUser" + strconv.Itoa(int(displayNameNum.Int64())),
			ID:            uuid.New().String(),
			Email:         "lub@dub.com",
			FirstName:     "test",
			LastName:      "case",
			AddressString: "https://usersTest.com",
			IsActive:      true,
			IsInternal:    false,
			Locale:        "en_US",
			OrgID:         orgID,
			DisplayName:   "FedRAMP" + strconv.Itoa(int(displayNameNum.Int64())),
			Type:          "User",
		})
	}

	return users, nil
}

func (ocm *SDKMock) GetAccountV3UsersBy(orgID string, q models.UserV3Query, _ models.UsersByBody) (models.Users, error) {
	users := models.Users{Users: []models.User{}}

	if orgID == "empty" {
		return users, nil
	}

	if orgID == "errorTest" {
		return users, fmt.Errorf("error retrieving V3 Users")
	}

	for i := q.Offset; i < q.Limit; i++ {
		displayNameNum, err := rand.Int(rand.Reader, big.NewInt(99-0))
		if err != nil {
			return users, err
		}

		users.AddUser(models.User{
			Username:      "TestUser" + strconv.Itoa(int(displayNameNum.Int64())),
			ID:            uuid.New().String(),
			Email:         "lub@dub.com",
			FirstName:     "test",
			LastName:      "case",
			AddressString: "https://usersTest.com",
			IsActive:      true,
			IsInternal:    false,
			Locale:        "en_US",
			OrgID:         orgID,
			DisplayName:   "FedRAMP" + strconv.Itoa(int(displayNameNum.Int64())),
			Type:          "User",
		})
	}

	return users, nil
}

func (ocm *SDKMock) CloseSdkConnection() {
	// nil
}
