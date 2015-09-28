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
	"os"
	"path/filepath"
	"strings"
)

// interface file defines the methods necessary for us to open a file for uploads.
type file interface {
	io.Closer
	io.Reader
}

// interface filesystem defines the filesystem methods we need to support file uploads.
type fileSystem interface {
	Open(name string) (file, error)
}

// osFS is an implementation of fileSystem that uses the disk.
type osFS struct{}

func (osFS) Open(name string) (file, error) { return os.Open(name) }

type medium struct {
	ApplicationId     string
	ApplicationSecret string
	AccessToken       string
	Host              string
	fs                fileSystem
}

// New returns a new Medium API client which can be used to make RPC requests.
func New(id, secret string) *medium {
	return &medium{
		ApplicationId:     id,
		ApplicationSecret: secret,
		Host:              "https://api.medium.com",
		fs:                osFS{},
	}
}

type clientRequest struct {
	method string
	path   string
	data   interface{}
	format string
}

// Error defines an error received when making a request to the API.
type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// defaultCode is the default error code for failures.
const defaultCode = -1

// Error returns a string representing the error, satisfying the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("medium: %s (%d)", e.Message, e.Code)
}

// payload defines a struct to represent payloads that are returned from Medium.
type envelope struct {
	Data   interface{} `json:"data"`
	Errors []Error     `json:"errors"`
}

// request makes a request to Medium's API
func (m *medium) request(cr clientRequest, result interface{}) error {
	f := cr.format
	if f == "" {
		f = "json"
	}

	var body []byte
	var ct string
	var err error
	switch f {
	case "json":
		body, err = json.Marshal(cr.data)
		if err != nil {
			return Error{fmt.Sprintf("Could not marshal JSON: %s", err.Error()), defaultCode}
		}
		ct = "application/json"

	case "form":
		switch d := cr.data.(type) {
		case string:
			body = []byte(d)
		case []byte:
			body = d
		default:
			return Error{"Invalid data passed to form", defaultCode}
		}
		ct = "application/x-www-form-urlencoded"

	case "file":
		uo, ok := cr.data.(UploadOptions)
		if !ok {
			return Error{fmt.Sprintf("Invalid upload data: %s", err.Error()), defaultCode}
		}
		file, err := m.fs.Open(uo.FilePath)
		if err != nil {
			return Error{fmt.Sprintf("Could not open file: %s", err.Error()), defaultCode}
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
			return Error{fmt.Sprintf("Could not create form part: %s", err.Error()), defaultCode}
		}
		_, err = io.Copy(part, file)
		if err != nil {
			return Error{fmt.Sprintf("Could not copy data: %s", err.Error()), defaultCode}
		}
		w.Close()

		body = b.Bytes()
		ct = w.FormDataContentType()

	default:
		return Error{fmt.Sprintf("Unknown format: %s", cr.format), defaultCode}
	}

	req, err := http.NewRequest(cr.method, fmt.Sprintf("%s%s", m.Host, cr.path), bytes.NewReader(body))
	if err != nil {
		return Error{fmt.Sprintf("Could not create request: %s", err.Error()), defaultCode}
	}

	req.Header.Add("Content-Type", ct)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Charset", "utf-8")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", m.AccessToken))

	cl := &http.Client{}
	r, err := cl.Do(req)
	if err != nil {
		return Error{fmt.Sprintf("Failed to make request: %s", err.Error()), defaultCode}
	}
	defer r.Body.Close()

	c, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return Error{fmt.Sprintf("Could not read response: %s", err.Error()), defaultCode}
	}

	return parseResponse(c, r.StatusCode, &result)
}

// parseResponse parses a response from Medium.
func parseResponse(content []byte, code int, result interface{}) error {
	env := envelope{}
	err := json.Unmarshal(content, &env)
	if err != nil {
		return Error{fmt.Sprintf("Could not parse response: %s", err.Error()), defaultCode}
	}
	if 200 <= code && code < 300 {
		if env.Data != nil {
			content, _ = json.Marshal(env.Data)
		}
		return json.Unmarshal(content, &result)
	} else {
		e := env.Errors[0]
		return Error{e.Message, e.Code}
	}
}

// Borrowed from multipart/writer.go
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
