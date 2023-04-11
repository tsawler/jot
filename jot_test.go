package jot

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJotGetTokenFromHeaderAndVerify(t *testing.T) {
	// create a test user; we need it to generate tokens
	testUser := User{
		ID:        1,
		FirstName: "Admin",
		LastName:  "User",
	}

	// create token pair
	tokens, _ := app.GenerateTokenPair(&testUser)

	// set up table tests
	tests := []struct {
		name          string
		token         string
		errorExpected bool
		setHeader     bool
	}{
		{"valid", fmt.Sprintf("Bearer %s", tokens.Token), false, true},
		{"valid expired", fmt.Sprintf("Bearer %s", expiredToken), true, true},
		{"no header", "", true, false},
		{"invalid", fmt.Sprintf("Bearer %s1", tokens.Token), true, true},
		{"empty header", "", true, true},
		{"no bearer", tokens.Token, true, true},
		{"not bearer", fmt.Sprintf("Bear %s", tokens.Token), true, true},
		{"three header parts", fmt.Sprintf("Bearer %s extratext", tokens.Token), true, true},
		{"wrong issuer", fmt.Sprintf("Bearer %s extratext", tokens.Token), true, true},
	}

	for _, e := range tests {
		// create a request
		req, _ := http.NewRequest("GET", "/", nil)

		// set authorization header, if appropriate
		if e.setHeader {
			req.Header.Set("Authorization", e.token)
		}

		// create a response recorder
		rr := httptest.NewRecorder()

		// try to verify token
		_, _, err := app.GetTokenFromHeaderAndVerify(rr, req)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: did not expect error, but got one - %v", e.name, err)
		}

		if err == nil && e.errorExpected {
			t.Errorf("%s: expected error, but did not get one", e.name)
		}
	}
}

func TestJotGetTokenFromHeaderAndVerifyWithBadIssuer(t *testing.T) {
	// create a test user
	testUser := User{
		ID:        1,
		FirstName: "Admin",
		LastName:  "User",
	}

	// save existing domain
	oldDomain := app.Issuer

	// set to other domain
	app.Issuer = "other.org"

	// issue token with other.org as issuer
	tokens, _ := app.GenerateTokenPair(&testUser)

	// set issuer back to example.com
	app.Issuer = oldDomain

	// create a request and set header
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.Token))

	// create a response recorder
	rr := httptest.NewRecorder()

	// try to verify token
	_, _, err := app.GetTokenFromHeaderAndVerify(rr, req)
	if err == nil {
		t.Error("should have error for bad issuer, but do not")
	}

}

func TestJotGetRefreshCookie(t *testing.T) {
	// create a test user
	testUser := User{
		ID:        1,
		FirstName: "Admin",
		LastName:  "User",
	}

	// generate tokens
	tokens, _ := app.GenerateTokenPair(&testUser)

	// get a refresh cookie
	c := app.GetRefreshCookie(tokens.RefreshToken)
	if !c.Expires.After(time.Now()) {
		t.Error("cookie expiration not set to future, and should be")
	}
}

func TestJotGetExpiredRefreshCookie(t *testing.T) {
	// call GetExpiredRefreshCookie
	c := app.GetExpiredRefreshCookie()
	if c.Expires.After(time.Now()) {
		t.Error("cookie expiration set to future, and should not be")
	}
}

func TestJotNew(t *testing.T) {
	var j = New("example.com")

	if j.CookieName != "refresh_token" {
		t.Error("refresh token name not expected value of `refresh_token`")
	}

}
