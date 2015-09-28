// Copyright 2015 A Medium Corporation

// Package medium provides a client for Medium's OAuth2 API.
package medium

import (
	"fmt"
)

// API defines the methods that can be performed with Medium's API.
type API interface {
	GetUser() (User, error)
	CreatePost(CreatePostOptions) (Post, error)
	UploadImage(UploadOptions) (Image, error)
}

// contentFormat enumerates the valid content formats for creating a post on Medium.
type contentFormat string

const (
	HTML contentFormat = "html"
)

// publishStatus enumerates the valid post statuses for creating a post on Medium.
type publishStatus string

const (
	Draft    publishStatus = "draft"
	Unlisted publishStatus = "unlisted"
	Public   publishStatus = "public"
)

// license enumerates the valid post licenses for creating a post on Medium.
type license string

const (
	AllRightsReserved license = "all-rights-reserved"
	CC40By            license = "cc-40-by"
	CC40BySA          license = "cc-40-by-sa"
	CC40ByND          license = "cc-40-by-nd"
	CC40ByNC          license = "cc-40-by-nc"
	CC40ByNCND        license = "cc-40-by-nc-nd"
	CC40ByNCSA        license = "cc-40-by-nc-sa"
	CC40Zero          license = "cc-40-zero"
	PublicDomain      license = "public-domain"
)

// CreatePostOptions defines the options for creating a post on Medium.
type CreatePostOptions struct {
	UserId        string        `json:"-"`
	Title         string        `json:"title"`
	Content       string        `json:"content"`
	ContentFormat contentFormat `json:"contentFormat"`
	Tags          []string      `json:"tags,omitempty"`
	CanonicalURL  string        `json:"canonicalUrl,omitempty"`
	PublishStatus publishStatus `json:"publishStatus,omitempty"`
	License       license       `json:"license,omitempty"`
}

// UploadOptions defines the options for uploading files to Medium.
type UploadOptions struct {
	FilePath    string
	ContentType string
	fieldName   string
}

// User defines a Medium user
type User struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	ImageURL string `json:"imageUrl"`
}

// Post defines a Medium post
type Post struct {
	Id           string        `json:"id"`
	Title        string        `json:"title"`
	AuthorId     string        `json:"authorId"`
	Tags         []string      `json:"tags"`
	URL          string        `json:"url"`
	CanonicalURL string        `json:"canonicalUrl"`
	PublishState publishStatus `json:"publishStatus"`
	License      license       `json:"license"`
	LicenseURL   string        `json:"licenseUrl"`
}

// Image defines a Medium image
type Image struct {
	URL string `json:"url"`
	MD5 string `json:"md5"`
}

// GetUser gets the profile identified by the current AccessToken.
// This requires AccessToken to have the BasicProfile scope.
func (m *medium) GetUser() (u User, err error) {
	r := clientRequest{
		method: "GET",
		path:   "/v1/me",
	}
	err = m.request(r, &u)
	return u, err
}

// CreatePost creates a post on the profile identified by the current AccessToken.
// This requires AccessToken to have the PublishPost scope.
func (m *medium) CreatePost(o CreatePostOptions) (p Post, err error) {
	r := clientRequest{
		method: "POST",
		path:   fmt.Sprintf("/v1/users/%s/posts", o.UserId),
		data:   o,
	}
	err = m.request(r, &p)
	return p, err
}

// UploadImage uploads an image to Medium.
// This requires AccessToken to have the UploadImage scope.
func (m *medium) UploadImage(o UploadOptions) (i Image, err error) {
	o.fieldName = "image"
	r := clientRequest{
		method: "POST",
		path:   fmt.Sprintf("/v1/images"),
		format: "file",
		data:   o,
	}
	err = m.request(r, &i)
	return i, err
}
