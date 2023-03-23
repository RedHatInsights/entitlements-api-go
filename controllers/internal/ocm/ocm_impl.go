package ocm

import (
	"context"
	"fmt"

	sdk "github.com/openshift-online/ocm-sdk-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/redhatinsights/mbop/internal/config"
	"github.com/redhatinsights/mbop/internal/models"
)

const OrganizationID = "organization.id"

type SDK struct {
	client *sdk.Connection
}

func (ocm *SDK) InitSdkConnection(ctx context.Context) error {
	// Create a logger that has the debug level enabled:
	logger, err := logging.NewGoLoggerBuilder().
		Debug(config.Get().Debug).
		Build()

	if err != nil {
		return err
	}

	ocm.client, err = sdk.NewConnectionBuilder().
		Logger(logger).

		// SA Auth:
		Client(config.Get().CognitoAppClientID, config.Get().CognitoAppClientSecret).

		// Offline Token Auth:
		// Tokens(<token>).

		// Oauth Token URL:
		TokenURL(config.Get().OauthTokenURL).

		// Route to hit for AMS:
		URL(config.Get().AmsURL).

		// SA Scopes:
		Scopes(config.Get().CognitoScope).
		BuildContext(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (ocm *SDK) GetUsers(usernames models.UserBody, q models.UserV1Query) (models.Users, error) {
	search := createSearchString(usernames)
	collection := ocm.client.AccountsMgmt().
		V1().
		Accounts().
		List().
		Parameter("fetchLabels", true).
		Search(search)

	collection = collection.Order(createQueryOrder(q))

	users := models.Users{Users: []models.User{}}
	usersResponse, err := collection.Send()
	if err != nil {
		return users, err
	}

	if usersResponse.Items().Empty() {
		return users, err
	}

	users = responseToUsers(usersResponse)

	return users, err
}

func (ocm *SDK) GetOrgAdmin(u []models.User) (models.OrgAdminResponse, error) {
	search := createOrgAdminSearchString(u)

	collection := ocm.client.AccountsMgmt().V1().RoleBindings()
	roleBindings, err := collection.List().Search(search).Send()

	orgAdminResponse := models.OrgAdminResponse{}
	if err != nil {
		return orgAdminResponse, err
	}

	if roleBindings.Items().Empty() {
		return orgAdminResponse, err
	}

	bindingSlice := roleBindings.Items().Slice()
	for _, binding := range bindingSlice {
		orgAdminResponse[binding.Account().ID()] = models.OrgAdmin{
			ID:         binding.Account().ID(),
			IsOrgAdmin: true,
		}
	}

	return orgAdminResponse, err
}

func (ocm *SDK) GetAccountV3Users(orgID string, q models.UserV3Query) (models.Users, error) {
	search := createAccountsV3UsersSearchString(orgID)

	collection := ocm.client.AccountsMgmt().V1().Accounts().List().Search(search)

	collection = collection.Order(createV3QueryOrder(q))
	collection = collection.Size(q.Limit)
	collection = collection.Page(q.Offset)

	users := models.Users{Users: []models.User{}}
	AccountV3UsersResponse, err := collection.Send()
	if err != nil {
		return users, err
	}

	users = responseToUsers(AccountV3UsersResponse)

	return users, err
}

func (ocm *SDK) GetAccountV3UsersBy(orgID string, q models.UserV3Query, body models.UsersByBody) (models.Users, error) {
	search := createAccountsV3UsersBySearchString(orgID, body)

	collection := ocm.client.AccountsMgmt().V1().Accounts().List().Search(search)

	collection = collection.Order(createV3QueryOrder(q))
	collection = collection.Size(q.Limit)
	collection = collection.Page(q.Offset)

	users := models.Users{Users: []models.User{}}
	AccountV3UsersResponse, err := collection.Send()
	if err != nil {
		return users, err
	}

	users = responseToUsers(AccountV3UsersResponse)

	return users, err
}

func (ocm *SDK) CloseSdkConnection() {
	ocm.client.Close()
}

func getIsInternal(user *v1.Account) bool {
	labels := user.Labels()
	for _, l := range labels {
		labelExists := l.Key() == config.Get().IsInternalLabel
		labelTruthy := l.Value() == "true"
		if labelExists && labelTruthy {
			return true
		}
	}
	return false
}

func responseToUsers(response *v1.AccountsListResponse) models.Users {
	users := models.Users{}
	items := response.Items().Slice()

	for i := range items {
		users.AddUser(models.User{
			Username:      items[i].Username(),
			ID:            items[i].ID(),
			Email:         items[i].Email(),
			FirstName:     items[i].FirstName(),
			LastName:      items[i].LastName(),
			AddressString: items[i].HREF(),
			IsActive:      true,
			IsInternal:    getIsInternal(items[i]),
			Locale:        "en_US",
			OrgID:         items[i].Organization().ID(),
			DisplayName:   items[i].Organization().Name(),
			Type:          items[i].Kind(),
		})
	}

	return users
}

func createSearchString(u models.UserBody) string {
	search := ""

	for i := range u.Users {
		if i > 0 {
			search += " or "
		}

		search += fmt.Sprintf("username='%s'", u.Users[i])
	}

	return search
}

func createOrgAdminSearchString(users []models.User) string {
	search := ""

	for i := range users {
		if i > 0 {
			search += " or "
		}

		search += fmt.Sprintf("account.id='%s' and role.id='OrganizationAdmin'", users[i].ID)
	}

	return search
}

func createAccountsV3UsersSearchString(orgID string) string {
	return fmt.Sprintf(OrganizationID+"='%s'", orgID)
}

func createAccountsV3UsersBySearchString(orgID string, body models.UsersByBody) string {
	search := createAccountsV3UsersSearchString(orgID)

	if body.EmailStartsWith != "" {
		search += fmt.Sprint(" and email like '" + body.EmailStartsWith + "%'")
	}

	if body.PrimaryEmail != "" {
		search += fmt.Sprintf(" and email='%s'", body.PrimaryEmail)
	}

	if body.PrincipalStartsWith != "" {
		search += fmt.Sprint(" and username like '" + body.PrincipalStartsWith + "%'")
	}

	return search
}

func createQueryOrder(q models.UserV1Query) string {
	order := ""

	if q.QueryBy != "" {
		order += q.QueryBy
	}

	if q.SortOrder != "" {
		order += fmt.Sprint(" " + q.SortOrder)
	}

	return order
}

var order = OrganizationID

func createV3QueryOrder(q models.UserV3Query) string {
	if q.SortOrder != "" {
		order += fmt.Sprint(" " + q.SortOrder)
	}

	return order
}
