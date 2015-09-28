// Copyright 2015 A Medium Corporation

// Package medium provides a client for Medium's OAuth2 API.
package medium

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	BasicProfile = "basicProfile"
	PublishPost  = "publishPost"
	UploadImage  = "uploadImage"
)

// type AccessToken defines credentials with which Medium's API may be accessed.
type AccessToken struct {
	TokenType    string   `json:"token_type"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	Scope        []string `json:"scope"`
	ExpiresAt    int64    `json:"expires_at"`
}

// GetAuthorizationUrl constructs the URL to which an application may send
// a user in order to acquire authorization.
func (m *medium) GetAuthorizationUrl(scope []string, state, redirectUrl string) string {
	v := url.Values{
		"client_id":     {m.ApplicationId},
		"scope":         {strings.Join(scope, ",")},
		"state":         {state},
		"response_type": {"code"},
		"redirect_uri":  {redirectUrl},
	}
	return fmt.Sprintf("https://medium.com/m/oauth/authorize?%s", v.Encode())
}

// ExchangeAuthorizationCode exchanges the supplied code for a long-lived access token.
func (m *medium) ExchangeAuthorizationCode(code, redirectUrl string) (at AccessToken, err error) {
	v := url.Values{
		"code":          {code},
		"client_id":     {m.ApplicationId},
		"client_secret": {m.ApplicationSecret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectUrl},
	}
	return m.acquireAccessToken(v)
}

// ExchangeRefreshToken exchanges the supplied refresh token for a new access token.
func (m *medium) ExchangeRefreshToken(rt string) (at AccessToken, err error) {
	v := url.Values{
		"refresh_token": {rt},
		"client_id":     {m.ApplicationId},
		"client_secret": {m.ApplicationSecret},
		"grant_type":    {"refresh_token"},
	}
	return m.acquireAccessToken(v)
}

// acquireAccessToken makes a request to Medium for an access token.
func (m *medium) acquireAccessToken(v url.Values) (at AccessToken, err error) {
	cr := clientRequest{
		method: "POST",
		path:   "/v1/tokens",
		format: "form",
		data:   v.Encode(),
	}
	err = m.request(cr, &at)

	// Set the access token on the service.
	if err != nil {
		m.AccessToken = at.AccessToken
	}
	return at, err
}
