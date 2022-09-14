[![Version](https://img.shields.io/badge/goversion-1.19.x-blue.svg)](https://golang.org)
<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/tsawler/jot/master/LICENSE)
![Tests](https://github.com/tsawler/jot/actions/workflows/tests.yml/badge.svg)


# Jot

A simple package to implement generating and verifying JWT tokens. It generates and verifies both auth tokens and 
refresh tokens.

Usage:

~~~Go
package main

import (
	"fmt"
	"github.com/tsawler/jot"
	"log"
	"net/http"
	"net/http/httptest"
	"time"
)

func main() {
	// create an instance of jot
	j := jot.Auth{
		Issuer:        "example.com",
		Audience:      "example.com",
		Secret:        "verysecretkey",
		TokenExpiry:   time.Minute * 15,
		RefreshExpiry: time.Hour * 24,
	}

	// set up a test user
	someUser := jot.User{
		ID:        1,
		FirstName: "John",
		LastName:  "Smith",
	}

	// generate tokens
	tokenPairs, _ := j.GenerateTokenPair(&someUser)
	log.Println("Token:", tokenPairs.Token)
	log.Println("Refresh Token:", tokenPairs.RefreshToken)

	// get a refresh token cookie
	cookie := j.GetRefreshCookie(tokenPairs.RefreshToken)
	log.Println("Cookie expiration:", cookie.Expires.UTC())

	// assuming that you are running a web app, you'll have a handler
	// that takes a request with the Authorization header set to
	// Bearer <some token>
	// where <some token> is a JWT token
	
	// let's build a request/response pair to send to GetTokenFromHeaderAndVerify
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/some-route", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenPairs.Token))

	// call GetTokenFromHeaderAndVerify
	_, _, err := j.GetTokenFromHeaderAndVerify(res, req)
	
	// print out validation results
	if err != nil {
		log.Println("Invalid token:", err.Error())
	} else {
		log.Println("Valid token!")
	}
}
~~~