package ss12000v2import

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/context/ctxhttp"

	"github.com/Sambruk/windermere/ss12000v2"
)

func getNewToken(ctx context.Context, clientId, clientSecret string) (string, time.Time, error) {
	const authServer = "https://skolid.se/connect/token"
	const safetyMargin = 5 * time.Minute

	form := make(url.Values)
	form.Add("grant_type", "client_credentials")
	form.Add("client_id", clientId)
	form.Add("client_secret", clientSecret)

	response, err := ctxhttp.PostForm(ctx, http.DefaultClient, authServer, form)

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to post form to %s: %s", authServer, err.Error())
	}

	if response.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("unexpected status code from %s: %d", authServer, response.StatusCode)
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	type answerType struct {
		AccessToken *string `json:"access_token,omitempty"`
		ExpiresIn   *int    `json:"expired_in,omitempty"`
	}
	var answer answerType

	err = json.Unmarshal(body, &answer)

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse response from %s: %s", authServer, err.Error())
	}

	if answer.AccessToken == nil {
		return "", time.Time{}, fmt.Errorf("no access token returned from %s", authServer)
	}

	expirationTime := time.Now()
	if answer.ExpiresIn != nil {
		expirationTime = time.Now().Add(time.Duration(*answer.ExpiresIn)*time.Second - safetyMargin)
	}

	return *answer.AccessToken, expirationTime, nil
}

func NewEduCloudClient(url, clientId, clientSecret string) (ss12000v2.ClientInterface, error) {
	token := ""
	var expirationTime time.Time
	apiKeyAdder := func(ctx context.Context, req *http.Request) error {
		if time.Now().After(expirationTime) {
			var err error
			token, expirationTime, err = getNewToken(ctx, clientId, clientSecret)
			if err != nil {
				return fmt.Errorf("failed to get new EduCloud token, clientId: %s, error: %s", clientId, err.Error())
			}
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
	return ss12000v2.NewClient(url, ss12000v2.WithRequestEditorFn(apiKeyAdder))
}
