// Copyright 2015 A Medium Corporation

// Package medium provides a client for Medium's OAuth2 API.
package medium

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Available scope options when requesting access to a user's Medium account.
const (
	ScopeBasicProfile Scope = "basicProfile"
	ScopePublishPost        = "publishPost"
	ScopeUploadImage        = "uploadImage"
	ScopeListPublications   = "listPublications"
)

// Content formats that are available when creating a post on Medium.
const (
	ContentFormatHTML     ContentFormat = "html"
	ContentFormatMarkdown               = "markdown"
)

// Publish statuses that are available when creating a post on Medium.
const (
	PublishStatusDraft    PublishStatus = "draft"
	PublishStatusUnlisted               = "unlisted"
	PublishStatusPublic                 = "public"
)

// Licenses that are available when creating a post on Medium.
const (
	LicenseAllRightsReserved License = "all-rights-reserved"
	LicenseCC40By                    = "cc-40-by"
	LicenseCC40BySA                  = "cc-40-by-sa"
	LicenseCC40ByND                  = "cc-40-by-nd"
	LicenseCC40ByNC                  = "cc-40-by-nc"
	LicenseCC40ByNCND                = "cc-40-by-nc-nd"
	LicenseCC40ByNCSA                = "cc-40-by-nc-sa"
	LicenseCC40Zero                  = "cc-40-zero"
	LicensePublicDomain              = "public-domain"
)

const (
	// host is the default host of Medium's API.
	host = "https://api.medium.com"
	// defaultTimeout is the default timeout duration used on HTTP requests.
	defaultTimeout = 5 * time.Second
	// defaultCode is the default error code for failures.
	defaultCode = -1
)

// formats used for marshalling data for requests.
const (
	formatJSON = "json"
	formatForm = "form"
	formatFile = "file"
)

// fileOpener defines the methods needed to support file uploads.
type fileOpener interface {
	Open(name string) (io.ReadCloser, error)
}

// CreatePostOptions defines the options for creating a post on Medium.
type CreatePostOptions struct {
	UserID        string        `json:"-"`
	Title         string        `json:"title"`
	Content       string        `json:"content"`
	ContentFormat ContentFormat `json:"contentFormat"`
	Tags          []string      `json:"tags,omitempty"`
	CanonicalURL  string        `json:"canonicalUrl,omitempty"`
	PublishStatus PublishStatus `json:"publishStatus,omitempty"`
	License       License       `json:"license,omitempty"`
}

// UploadOptions defines the options for uploading files to Medium.
type UploadOptions struct {
	FilePath    string
	ContentType string
	fieldName   string
}

// AccessToken defines credentials with which Medium's API may be accessed.
type AccessToken struct {
	TokenType    string   `json:"token_type"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	Scope        []string `json:"scope"`
	ExpiresAt    int64    `json:"expires_at"`
}

// User defines a Medium user
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	ImageURL string `json:"imageUrl"`
}

// Publications inherit all Medium user publications
type Publications struct {
	Data []Publication `json:"data"`
}

// Publication defines a Medium user publication
type Publication struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	ImageURL    string `json:"imageUrl"`
}

// Contributors inherit all Medium publication contributors
type Contributors struct {
	Data []Contributor `json:"data"`
}

// Contributor defines a Medium publication contributor
type Contributor struct {
	PublicationID string `json:"publicationID"`
	UserID        string `json:"userID"`
	Role          string `json:"role"`
}

// Post defines a Medium post
type Post struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	AuthorID     string        `json:"authorId"`
	Tags         []string      `json:"tags"`
	URL          string        `json:"url"`
	CanonicalURL string        `json:"canonicalUrl"`
	PublishState PublishStatus `json:"publishStatus"`
	License      License       `json:"license"`
	LicenseURL   string        `json:"licenseUrl"`
}

// Image defines a Medium image
type Image struct {
	URL string `json:"url"`
	MD5 string `json:"md5"`
}

// Error defines an error received when making a request to the API.
type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Error returns a string representing the error, satisfying the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("medium: %s (%d)", e.Message, e.Code)
}

// Medium defines the Medium client.
type Medium struct {
	ApplicationID     string
	ApplicationSecret string
	AccessToken       string
	Host              string
	Timeout           time.Duration
	Transport         http.RoundTripper
	fs                fileOpener
}

// NewClient returns a new Medium API client which can be used to make RPC requests.
func NewClient(id, secret string) *Medium {
	return &Medium{
		ApplicationID:     id,
		ApplicationSecret: secret,
		Host:              host,
		Timeout:           defaultTimeout,
		Transport:         http.DefaultTransport,
		fs:                osFS{},
	}
}

// NewClientWithAccessToken returns a new Medium API client which can be used to make RPC requests.
func NewClientWithAccessToken(accessToken string) *Medium {
	return &Medium{
		AccessToken: accessToken,
		Host:        host,
		fs:          osFS{},
	}
}

// GetAuthorizationURL returns the URL to which an application may send
// a user in order to acquire authorization.
func (m *Medium) GetAuthorizationURL(state, redirectURL string, scopes ...Scope) string {
	s := make([]string, len(scopes))
	for i, scp := range scopes {
		s[i] = string(scp)
	}
	v := url.Values{
		"client_id":     {m.ApplicationID},
		"scope":         {strings.Join(s, ",")},
		"state":         {state},
		"response_type": {"code"},
		"redirect_uri":  {redirectURL},
	}
	return fmt.Sprintf("https://medium.com/m/oauth/authorize?%s", v.Encode())
}

// ExchangeAuthorizationCode exchanges the supplied code for a long-lived access token.
func (m *Medium) ExchangeAuthorizationCode(code, redirectURL string) (AccessToken, error) {
	v := url.Values{
		"code":          {code},
		"client_id":     {m.ApplicationID},
		"client_secret": {m.ApplicationSecret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURL},
	}
	return m.acquireAccessToken(v)
}

// ExchangeRefreshToken exchanges the supplied refresh token for a new access token.
func (m *Medium) ExchangeRefreshToken(rt string) (AccessToken, error) {
	v := url.Values{
		"refresh_token": {rt},
		"client_id":     {m.ApplicationID},
		"client_secret": {m.ApplicationSecret},
		"grant_type":    {"refresh_token"},
	}
	return m.acquireAccessToken(v)
}

// GetUser gets the profile identified by the current AccessToken.
// It will get the specified user or the current user if userID is empty.
// This requires m.AccessToken to have the BasicProfile scope.
func (m *Medium) GetUser(userID string) (*User, error) {
	var r clientRequest
	if userID == "" {
		r = clientRequest{
			method: "GET",
			path:   "/v1/me",
		}
	} else {
		r = clientRequest{
			method: "GET",
			path:   fmt.Sprintf("/v1/%s", userID),
		}
	}
	u := &User{}
	err := m.request(r, u)
	return u, err
}

// GetUserPublications gets user publications by the current AccessToken.
// This requires m.AccessToken to have the BasicPublications scope.
func (m *Medium) GetUserPublications(userID string) (*Publications, error) {
	r := clientRequest{
		method: "GET",
		path:   fmt.Sprintf("/v1/users/%s/publications", userID),
	}
	p := &Publications{}
	err := m.request(r, p)
	return p, err
}

// GetPublicationContributors gets contributors for givaen a publication
// by the current AccessToken.
// This requires m.AccessToken to have the BasicPublications scope.
func (m *Medium) GetPublicationContributors(publicationID string) (*Contributors, error) {
	r := clientRequest{
		method: "GET",
		path:   fmt.Sprintf("/v1/publications/%s/contributors", publicationID),
	}
	p := &Contributors{}
	err := m.request(r, p)
	return p, err
}

// CreatePost creates a post on the profile identified by the current AccessToken.
// This requires m.AccessToken to have the PublishPost scope.
func (m *Medium) CreatePost(o CreatePostOptions) (*Post, error) {
	r := clientRequest{
		method: "POST",
		path:   fmt.Sprintf("/v1/users/%s/posts", o.UserID),
		data:   o,
	}
	p := &Post{}
	err := m.request(r, p)
	return p, err
}

// UploadImage uploads an image to Medium.
// This requires m.AccessToken to have the UploadImage scope.
func (m *Medium) UploadImage(o UploadOptions) (*Image, error) {
	o.fieldName = "image"
	r := clientRequest{
		method: "POST",
		path:   fmt.Sprintf("/v1/images"),
		format: formatFile,
		data:   o,
	}
	i := &Image{}
	err := m.request(r, i)
	return i, err
}

// generateJSONRequestData returns the body and content type for a JSON request.
func (m *Medium) generateJSONRequestData(cr clientRequest) ([]byte, string, error) {
	body, err := json.Marshal(cr.data)
	if err != nil {
		return nil, "", Error{fmt.Sprintf("Could not marshal JSON: %s", err), defaultCode}
	}
	return body, "application/json", nil
}

// generateFormRequestData returns the body and content type for a form data request.
func (m *Medium) generateFormRequestData(cr clientRequest) ([]byte, string, error) {
	var body []byte
	switch d := cr.data.(type) {
	case string:
		body = []byte(d)
	case []byte:
		body = d
	default:
		return nil, "", Error{"Invalid data passed for form request", defaultCode}
	}
	return body, "application/x-www-form-urlencoded", nil
}

// generateFileRequestData returns the body and content type for a file upload request.
func (m *Medium) generateFileRequestData(cr clientRequest) ([]byte, string, error) {
	uo, ok := cr.data.(UploadOptions)
	if !ok {
		return nil, "", Error{"Invalid data passed for file upload", defaultCode}
	}
	file, err := m.fs.Open(uo.FilePath)
	if err != nil {
		return nil, "", Error{fmt.Sprintf("Could not open file: %s", err), defaultCode}
	}
	defer file.Close()

	// Create a form part
	b := bytes.Buffer{}
	w := multipart.NewWriter(&b)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		escapeQuotes(uo.fieldName), escapeQuotes(filepath.Base(uo.FilePath))))
	h.Set("Content-Type", uo.ContentType)
	part, err := w.CreatePart(h)
	if err != nil {
		return nil, "", Error{fmt.Sprintf("Could not create form part: %s", err), defaultCode}
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, "", Error{fmt.Sprintf("Could not copy data: %s", err), defaultCode}
	}
	w.Close()

	return b.Bytes(), w.FormDataContentType(), nil
}

// request makes a request to Medium's API
func (m *Medium) request(cr clientRequest, result interface{}) error {
	f := cr.format
	if f == "" {
		f = formatJSON
	}

	// Get the body and content type.
	var g requestDataGenerator
	switch f {
	case formatJSON:
		g = m.generateJSONRequestData
	case formatForm:
		g = m.generateFormRequestData
	case formatFile:
		g = m.generateFileRequestData
	default:
		return Error{fmt.Sprintf("Unknown format: %s", cr.format), defaultCode}
	}
	body, ct, err := g(cr)
	if err != nil {
		return err
	}

	// Construct the request
	req, err := http.NewRequest(cr.method, m.Host+cr.path, bytes.NewReader(body))
	if err != nil {
		return Error{fmt.Sprintf("Could not create request: %s", err), defaultCode}
	}

	req.Header.Add("Content-Type", ct)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Charset", "utf-8")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", m.AccessToken))

	// Create the HTTP client
	client := &http.Client{
		Transport: m.Transport,
		Timeout:   m.Timeout,
	}

	// Make the request
	res, err := client.Do(req)
	if err != nil {
		return Error{fmt.Sprintf("Failed to make request: %s", err), defaultCode}
	}
	defer res.Body.Close()

	// Parse the response
	c, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Error{fmt.Sprintf("Could not read response: %s", err), defaultCode}
	}

	var env envelope
	if err := json.Unmarshal(c, &env); err != nil {
		return Error{fmt.Sprintf("Could not parse response: %s", err), defaultCode}
	}

	if http.StatusOK <= res.StatusCode && res.StatusCode < http.StatusMultipleChoices {
		if env.Data != nil {
			c, _ = json.Marshal(env.Data)
		}
		return json.Unmarshal(c, &result)
	}
	e := env.Errors[0]
	return Error{e.Message, e.Code}
}

// acquireAccessToken makes a request to Medium for an access token.
func (m *Medium) acquireAccessToken(v url.Values) (AccessToken, error) {
	cr := clientRequest{
		method: "POST",
		path:   "/v1/tokens",
		format: formatForm,
		data:   v.Encode(),
	}
	at := AccessToken{}
	err := m.request(cr, &at)

	// Set the access token on the service.
	if err == nil {
		m.AccessToken = at.AccessToken
	}
	return at, err
}

type ContentFormat string
type PublishStatus string
type License string
type Scope string

// clientRequest defines information that can be used to make a request to Medium.
type clientRequest struct {
	method string
	path   string
	data   interface{}
	format string
}

// payload defines a struct to represent payloads that are returned from Medium.
type envelope struct {
	Data   interface{} `json:"data"`
	Errors []Error     `json:"errors,omitempty"`
}

// osFS is an implementation of fileOpener that uses the disk.
type osFS struct{}

// Open opens a file from disk.
func (osFS) Open(name string) (io.ReadCloser, error) { return os.Open(name) }

// requestDataGenerator defines a function that can generate request data.
type requestDataGenerator func(cr clientRequest) ([]byte, string, error)

// Borrowed from multipart/writer.go
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// escapeQuotes returns the supplied string with quotes escaped.
func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
