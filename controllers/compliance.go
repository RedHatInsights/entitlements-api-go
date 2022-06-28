package controllers

import (
	"bytes"
	"encoding/json"
	"github.com/RedHatInsights/entitlements-api-go/config"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

var screeningPathV1 = "/v1/screening"

type User struct {
	Id    string `json:"id"`
	Login string `json:"login"`
}

type Account struct {
	Primary bool `json:"primary"`
}

type ComplianceScreeningRequest struct {
	User    *User    `json:"user"`
	Account *Account `json:"account"`
}

func Compliance() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var client = getClient()

		url := config.GetConfig().Options.GetString(config.Keys.ComplianceHost) + screeningPathV1
		body := ComplianceScreeningRequest{
			User:    &User{Id: "1234"},
			Account: &Account{Primary: true},
		}
		marshalledBody, err := json.Marshal(body)

		if err != nil {
			l.Log.WithFields(logrus.Fields{"error": err}).Error("Unexpected error while marshalling JSON data to send to Compliance Service")
			return
		}

		complianceReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(marshalledBody))

		if err != nil {
			l.Log.WithFields(logrus.Fields{"error": err}).Error("Unexpected error while creating request to Compliance Service")
			return
		}

		complianceReq.Header.Add("accept", "application/json;charset=UTF-8")
		complianceReq.Header.Add("Content-Type", "application/json;charset=UTF-8")

		resp, err := client.Do(complianceReq)

		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(respBody))
	}
}
