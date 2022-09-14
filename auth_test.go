package jot

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_app_getTokenFromHeaderAndVerify(t *testing.T) {
	testUser := User{
		ID:        1,
		FirstName: "Admin",
		LastName:  "User",
	}

	tokens, _ := app.GenerateTokenPair(&testUser)

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
		req, _ := http.NewRequest("GET", "/", nil)
		if e.setHeader {
			req.Header.Set("Authorization", e.token)
		}

		rr := httptest.NewRecorder()

		_, _, err := app.GetTokenFromHeaderAndVerify(rr, req)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: did not expect error, but got one - %v", e.name, err)
		}

		if err == nil && e.errorExpected {
			t.Errorf("%s: expected error, but did not get one", e.name)
		}
	}
}

func Test_app_getTokenFromHeaderAndVerifyWithBadIssuer(t *testing.T) {
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
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.Token))

	rr := httptest.NewRecorder()

	_, _, err := app.GetTokenFromHeaderAndVerify(rr, req)
	if err == nil {
		t.Error("should have error for bad issuer, but do not")
	}

}
