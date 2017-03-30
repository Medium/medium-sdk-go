// Copyright 2015 A Medium Corporation.

package medium

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

// fakeFS is a filesystem that works in memory.
type fakeFS struct{}

func (fakeFS) Open(name string) (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader([]byte("contents"))), nil
}

type apiTest struct {
	token       string
	fn          interface{}
	payload     interface{}
	method      string
	path        string
	contentType string
	bodyPattern string
}

var m = NewClient("clientId", "clientSecret")

var apiTests = []apiTest{
	{"token", m.GetUser, "",
		"GET", "/v1/me", "application/json",
		"null"},
	{"token", m.GetUser, "@dummyUser",
		"GET", "/v1/@dummyUser", "application/json",
		"null"},
	{"token", m.GetUserPublications, "@dummyUser",
		"GET", "/v1/users/@dummyUser/publications", "application/json",
		"null"},
	{"token", m.GetPublicationContributors, "b45573563f5a",
		"GET", "/v1/publications/b45573563f5a/contributors", "application/json",
		"null"},
	{"token", m.CreatePost, CreatePostOptions{UserID: "42", Title: "Title", Content: "Yo", ContentFormat: "html"},
		"POST", "/v1/users/42/posts", "application/json",
		`{"title":"Title","content":"Yo","contentFormat":"html"}`},
	{"token", m.UploadImage, UploadOptions{FilePath: "/fake/file.png", ContentType: "image/png"},
		"POST", "/v1/images", "multipart/form-data.*",
		`^--[a-z0-9]+\r\n(Content-Disposition: form-data; name="image"; filename="file.png"|Content-Type: image/png)\r\n(Content-Disposition: form-data; name="image"; filename="file.png"|Content-Type: image/png)\r\n\r\ncontents\r\n--[a-z0-9]+--\r\n$`},
}

// TestAPIMethods tests that http requests are constructed correctly.
func TestAPIMethods(t *testing.T) {
	m.fs = fakeFS{}
	var body []byte
	var req *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req = r
		body, _ = ioutil.ReadAll(req.Body)
	}))
	defer ts.Close()
	m.Host = ts.URL

	for _, tt := range apiTests {
		m.AccessToken = tt.token

		f := reflect.ValueOf(tt.fn)
		var pl []reflect.Value
		if tt.payload != nil {
			pl = append(pl, reflect.ValueOf(tt.payload))
		}
		f.Call(pl)

		// Test request was correctly formed.
		assertEqual(t, req.Header.Get("Authorization"), fmt.Sprintf("Bearer %s", tt.token))
		assertEqual(t, req.Header.Get("Accept"), "application/json")
		assertMatch(t, req.Header.Get("Content-Type"), tt.contentType)
		assertEqual(t, req.Method, tt.method)
		assertEqual(t, req.URL.Path, tt.path)
		assertMatch(t, string(body), tt.bodyPattern)
	}
}

// TestAPITimeout verifies that HTTP timeouts work
func TestAPITimeout(t *testing.T) {
	m.AccessToken = "token"
	m.Timeout = 1 * time.Millisecond
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(m.Timeout * 2) // sleep longer than timeout
		fmt.Fprintln(w, "null")
	}))
	defer ts.Close()
	m.Host = ts.URL

	_, err := m.GetUser("")
	if err == nil {
		t.Errorf("Expected HTTP timeout error, but call succeeded")
	} else if !strings.Contains(err.Error(), "Client.Timeout exceeded") {
		// go1.4 doesn't set the timeout error but closes the connection.
		if !strings.Contains(err.Error(), "use of closed network connection") {
			t.Errorf("Expected HTTP timeout error, got %s", err)
		}
	}
}

func assertEqual(t *testing.T, actual, expected interface{}) {
	if actual != expected {
		t.Errorf("Expected %#v, got %#v", expected, actual)
	}
}

func assertMatch(t *testing.T, actual, pattern string) {
	re := regexp.MustCompile(pattern)
	if !re.MatchString(actual) {
		t.Errorf("Expected to match %#v, got %#v", pattern, actual)
	}
}
