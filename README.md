# Medium SDK for Go

This repository contains the open source SDK for integrating [Medium](https://medium.com)'s OAuth2 API into your Go app.

Install
-------

    go get https://github.com/majelbstoat/medium-sdk-go

Usage
-----

Create a client, then call commands on it.


```go
  import (
    medium "github.com/majelbstoat/medium-sdk-go"
  )
  
  m := medium.New("YOUR_APPLICATION_ID", "YOUR_APPLICATION_SECRET")
  url := m.GetAuthorizationUrl([]string{medium.BasicProfile, medium.PublishPost},
      "https://yoursite.com/callback/medium")
  
  // (Send the user to the URL to acquire an authorization code.)

  at, err := m.ExchangeAuthorizationCode("YOUR_AUTHORIZATION_CODE", "https://yoursite.com/callback/medium")
  u, err := m.GetUser()
  p, err := m.CreatePost(medium.CreatePostOptions{
    UserId: u.Id,
    Title: "Title",
    Content: "<h2>Title</h2><p>Content</p>",
    ContentFormat: medium.HTML,
    PublishStatus: medium.Draft
  })
  
  // (Assuming no errors, the created post now lives at p.url.)
```

Contributions
-------------

Questions, comments, bug reports and pull requests are all welcomed, especially those that make the code more idiomatic.

Author
------

[Jamie Talbot](https://github.com/majelbstoat), supported by
[Medium](https://medium.com).

License
-------

Copyright 2015 [A Medium Corporation](https://medium.com)

Licensed under Apache License Version 2.0.  Details in the attached LICENSE
file.

