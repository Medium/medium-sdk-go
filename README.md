# Medium SDK for Go

This repository contains the open source SDK for integrating [Medium](https://medium.com)'s OAuth2 API into your Go app.

Install
-------

    go get github.com/Medium/medium-sdk-go

Usage
-----

Create a client, then call commands on it.

```go
package main

import (
	medium "github.com/medium/medium-sdk-go"
	"log"
)

func main() {
	// Go to https://medium.com/me/applications to get your applicationId and applicationSecret.
	m := medium.NewClient("YOUR_APPLICATION_ID", "YOUR_APPLICATION_SECRET")

	// Build the URL where you can send the user to obtain an authorization code.
	url := m.GetAuthorizationURL("secretstate", "https://yoursite.com/callback/medium",
        medium.ScopeBasicProfile, medium.ScopePublishPost)

	// (Send the user to the authorization URL to obtain an authorization code.)

	// Exchange the authorization code for an access token.
	at, err := m.ExchangeAuthorizationCode("YOUR_AUTHORIZATION_CODE", "https://yoursite.com/callback/medium")
	if err != nil {
		log.Fatal(err)
	}

	// The access token is automatically set on the client for you after
	// a successful exchange, but if you already have a token, you can set it
	// directly.
	m.AccessToken = at.AccessToken

	// If you have a self-issued access token, you can skip these steps and
	// create a new client directly:
	m2 := medium.NewClientWithAccessToken("SELF_ISSUED_ACCESS_TOKEN")

	// Get profile details of the user identified by the access token.
	// Empty string mean current user, otherwise you need to indicate
	// the user id (alphanumeric string with 65 chars)
	u, err := m2.GetUser("")
	if err != nil {
		log.Fatal(err)
	}

	// Create a draft post.
	p, err := m.CreatePost(medium.CreatePostOptions{
		UserID:        u.ID,
		Title:         "Title",
		Content:       "<h2>Title</h2><p>Content</p>",
		ContentFormat: medium.ContentFormatHTML,
		PublishStatus: medium.PublishStatusDraft,
	})
	if err != nil {
		log.Fatal(err)
	}

	// When your access token expires, use the refresh token to get a new one.
	nt, err := m.ExchangeRefreshToken(at.RefreshToken)
	if err != nil {
		log.Fatal(err)
	}

	// Confirm everything went ok. p.URL has the location of the created post.
	log.Println(url, at, u, p, nt)
}
```

Contributing
------------

Questions, comments, bug reports, and pull requests are all welcomed. If you haven't contributed to a Medium project before please head over to the [Open Source Project](https://github.com/Medium/opensource#note-to-external-contributors) and fill out an OCLA (it should be pretty painless).

Authors
-------

[Jamie Talbot](https://github.com/majelbstoat)

[Dan Pupius](https://github.com/dpup)

[Andrew Bonventre](https://github.com/andybons)

License
-------

Copyright 2015 [A Medium Corporation](https://medium.com)

Licensed under Apache License Version 2.0.  Details in the attached LICENSE
file.
