package jot

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"strings"
	"time"
)

// Auth is the type used to instantiate this package.
type Auth struct {
	Issuer        string
	Audience      string
	Secret        string
	Domain        string
	TokenExpiry   time.Duration
	RefreshExpiry time.Duration
}

// User is a generic type used to hold the minimal amount of data
// we require in order to issue tokens.
type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// TokenPairs is the type used to generate JSON containing the
// JWT token and the refresh token.
type TokenPairs struct {
	Token        string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Claims is the type used to describe the claims in a given token.
type Claims struct {
	jwt.RegisteredClaims
}

// GetTokenFromHeaderAndVerify extracts a token from the Authorization header, verifies it, and returns the
// token, the claims, and error, if any.
func (j *Auth) GetTokenFromHeaderAndVerify(w http.ResponseWriter, r *http.Request) (string, *Claims, error) {
	// add a header (as we should)
	w.Header().Add("Vary", "Authorization")

	// get the Authorization header
	authHeader := r.Header.Get("Authorization")

	// sanity check
	if authHeader == "" {
		return "", nil, errors.New("no auth header")
	}

	// split the header up on spaces
	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 {
		return "", nil, errors.New("invalid auth header")
	}

	// check to see if we have the word "Bearer" in the right spot (we should)
	if headerParts[0] != "Bearer" {
		return "", nil, errors.New("unauthorized - no bearer")
	}

	// get the actual token
	token := headerParts[1]

	// declare an empty Claims variable
	claims := &Claims{}

	// parse the token with our claims (we read into claims), using our secret (from the receiver)
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		// validate the signing algorithm is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.Secret), nil
	})

	// Check for errors. Note that this catches expired tokens as well.
	if err != nil {
		// return an easy to spot error if the token is expired
		if strings.HasPrefix(err.Error(), "token is expired by") {
			return "", nil, errors.New("expired token")
		}
		return "", nil, err
	}

	// make sure we issued this token
	if claims.Issuer != j.Issuer {
		// we did not issue this token
		return "", nil, errors.New("incorrect issuer")
	}

	// if we get this far, the token is valid, so we return it, along with the claims
	return token, claims, nil
}

func (j *Auth) GenerateTokenPair(user *User) (TokenPairs, error) {
	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	// set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	claims["sub"] = fmt.Sprint(user.ID)
	claims["aud"] = j.Audience
	claims["iss"] = j.Issuer
	claims["iat"] = time.Now().UTC().Unix()

	// set expiry; should be short!
	claims["exp"] = time.Now().UTC().Add(j.TokenExpiry).Unix()

	// create signed token
	signedAccessToken, err := token.SignedString([]byte(j.Secret))
	if err != nil {
		return TokenPairs{}, err
	}

	// create refresh token and set claims (just subject and expiry)
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshTokenClaims := refreshToken.Claims.(jwt.MapClaims)
	refreshTokenClaims["sub"] = fmt.Sprint(user.ID)
	refreshTokenClaims["iat"] = time.Now().UTC().Unix()
	// set expiry; must be longer than JWT token expiry!
	refreshTokenClaims["exp"] = time.Now().UTC().Add(j.RefreshExpiry).Unix()

	// create signed refresh token
	signedRefreshToken, err := refreshToken.SignedString([]byte(j.Secret))
	if err != nil {
		return TokenPairs{}, err
	}

	// create token pairs and populate with signed tokens
	var tokenPairs = TokenPairs{
		Token:        signedAccessToken,
		RefreshToken: signedRefreshToken,
	}

	// return the token pair, and no error
	return tokenPairs, nil
}

// GetRefreshCookie returns a cookie containing the refresh token. Note that
// the cookie is http only, secure, and set to same site strict mode.
func (j *Auth) GetRefreshCookie(refreshToken string) *http.Cookie {
	c := &http.Cookie{
		Name:     "__Host-refresh_token",
		Path:     "/",
		Value:    refreshToken,
		Expires:  time.Now().Add(j.RefreshExpiry),
		MaxAge:   int(j.RefreshExpiry.Seconds()),
		SameSite: http.SameSiteStrictMode,
		Domain:   j.Domain,
		HttpOnly: true,
		Secure:   true,
	}

	return c
}
