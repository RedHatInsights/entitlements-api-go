package securitylog

import (
	"os"

	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"github.com/sirupsen/logrus"
)

const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)

func Fields(action, resourceType, resourceID, outcome string, principal map[string]string) logrus.Fields {
	return logrus.Fields{
		"action":        action,
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"outcome":       outcome,
		"principal":     principal,
	}
}

func FieldsFromIdentity(id identity.Identity, action, resourceType, resourceID, outcome string) logrus.Fields {
	return Fields(action, resourceType, resourceID, outcome, PrincipalFromIdentity(id))
}

func PrincipalFromIdentity(id identity.Identity) map[string]string {
	principal := map[string]string{}

	if id.Internal.OrgID != "" {
		principal["org_id"] = id.Internal.OrgID
	}

	if id.AccountNumber != "" {
		principal["account_number"] = id.AccountNumber
	}

	if id.ServiceAccount != nil {
		principal["type"] = "service_account"
		if id.ServiceAccount.Username != "" {
			principal["service_account_id"] = id.ServiceAccount.Username
		}
		if id.ServiceAccount.ClientId != "" {
			principal["client_id"] = id.ServiceAccount.ClientId
		}
		return principal
	}

	if id.User != nil {
		principal["type"] = "user"
		if id.User.Username != "" {
			principal["user_id"] = id.User.Username
		}
		return principal
	}

	if id.Type != "" {
		principal["type"] = id.Type
	} else {
		principal["type"] = "unknown"
	}

	return principal
}

func UnknownPrincipal() map[string]string {
	return map[string]string{"type": "unknown"}
}

func ProcessPrincipal(service string) map[string]string {
	return map[string]string{"type": "system", "service_account_id": service}
}

func ResourceIDOrFallback(value, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}

func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.Out = os.Stdout
	log.Level = logrus.InfoLevel
	log.ReportCaller = true
	log.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "ts",
			logrus.FieldKeyFunc:  "caller",
			logrus.FieldKeyLevel: "logLevel",
			logrus.FieldKeyMsg:   "msg",
		},
	})
	return log
}
