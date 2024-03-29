package jot

import (
	"github.com/golang-jwt/jwt/v5"
	"os"
	"testing"
	"time"
)

var app Auth
var expiredToken string

func TestMain(m *testing.M) {
	app = Auth{
		Issuer:        "example.com",
		Audience:      "example.com",
		Secret:        "verysecret",
		TokenExpiry:   time.Minute * 15,
		RefreshExpiry: time.Hour * 24,
		CookieDomain:  "localhost",
	}

	// generate a token
	token := jwt.New(jwt.SigningMethodHS256)

	// set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = "John Doe"
	claims["sub"] = "1"
	claims["admin"] = true
	claims["aud"] = "example.com"
	claims["iss"] = "example.com"
	// we'll set expires to the past, so we can have an expired token
	expires := time.Now().UTC().Add(time.Hour * 100 * -1)
	claims["exp"] = expires.Unix()

	// generate an expired token
	expiredToken, _ = token.SignedString([]byte(app.Secret))

	os.Exit(m.Run())
}
