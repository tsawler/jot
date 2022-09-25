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
	Issuer        string        // Who issues the token, e.g. company.com.
	Audience      string        // Who is the token for, e.g. company.com.
	Secret        string        // A strong secret, used to sign the tokens.
	TokenExpiry   time.Duration // When does the token expire, e.g. time.Minute * 15.
	RefreshExpiry time.Duration // When does the refresh token expire, e.g. time.Hour * 24.
	CookieDomain  string        // The domain, for refresh cookies.
	CookiePath    string        // The path, for refresh cookies.
	CookieName    string        // The name of the refresh token cookie.
}

// User is a generic type used to hold the minimal amount of data we require in order to issue tokens.
type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// TokenPairs is the type used to generate JSON containing the JWT token and the refresh token.
type TokenPairs struct {
	Token        string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Claims is the type used to describe the claims in a given token.
type Claims struct {
	jwt.RegisteredClaims
}

// New returns an instance of Auth, with sensible defaults where possible. Naturally,
// any of defaults can be overridden, if necessary.
func New(d string) Auth {
	return Auth{
		Issuer:        d,
		Audience:      d,
		TokenExpiry:   time.Minute * 15,
		RefreshExpiry: time.Hour * 24,
		CookieName:    "__Host-refresh_token",
		CookiePath:    "/",
		CookieDomain:  d,
	}
}

// GetTokenFromHeaderAndVerify extracts a token from the Authorization header, verifies it, and returns the
// token, the claims, and error, if any.
func (j *Auth) GetTokenFromHeaderAndVerify(w http.ResponseWriter, r *http.Request) (string, *Claims, error) {
	w.Header().Add("Vary", "Authorization")

	// Get the Authorization header.
	authHeader := r.Header.Get("Authorization")

	// Sanity check.
	if authHeader == "" {
		return "", nil, errors.New("no auth header")
	}

	// Split the header up on spaces.
	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 {
		return "", nil, errors.New("invalid auth header")
	}

	// Check to see if we have the word "Bearer" in the right spot (we should).
	if headerParts[0] != "Bearer" {
		return "", nil, errors.New("unauthorized - no bearer")
	}

	// Get the actual token.
	token := headerParts[1]

	// Declare an empty Claims variable.
	claims := &Claims{}

	// Parse the token with our claims (we read into claims), using our secret (from the receiver).
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

	// Make sure we issued this token.
	if claims.Issuer != j.Issuer {
		// we did not issue this token
		return "", nil, errors.New("incorrect issuer")
	}

	// If we get this far, the token is valid, so we return it, along with the claims.
	return token, claims, nil
}

// GenerateTokenPair takes a user of type jot.User and attempts to generate a pair of tokens for that user
// (jwt and refresh tokens).
func (j *Auth) GenerateTokenPair(user *User) (TokenPairs, error) {
	// Create token.
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims.
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	claims["sub"] = fmt.Sprint(user.ID)
	claims["aud"] = j.Audience
	claims["iss"] = j.Issuer
	claims["iat"] = time.Now().UTC().Unix()
	claims["typ"] = "JWT"

	// Set expiry; should be short!
	claims["exp"] = time.Now().UTC().Add(j.TokenExpiry).Unix()

	// Create signed token.
	signedAccessToken, err := token.SignedString([]byte(j.Secret))
	if err != nil {
		return TokenPairs{}, err
	}

	// Create refresh token and set claims (just subject and expiry).
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshTokenClaims := refreshToken.Claims.(jwt.MapClaims)
	refreshTokenClaims["sub"] = fmt.Sprint(user.ID)
	refreshTokenClaims["iat"] = time.Now().UTC().Unix()

	// Set expiry; must be longer than JWT token expiry!
	refreshTokenClaims["exp"] = time.Now().UTC().Add(j.RefreshExpiry).Unix()

	// Create signed refresh token.
	signedRefreshToken, err := refreshToken.SignedString([]byte(j.Secret))
	if err != nil {
		return TokenPairs{}, err
	}

	// Create token pairs and populate with signed tokens.
	var tokenPairs = TokenPairs{
		Token:        signedAccessToken,
		RefreshToken: signedRefreshToken,
	}

	// Return the token pair, and no error.
	return tokenPairs, nil
}

// GetRefreshCookie returns a cookie containing the refresh token. Note that the cookie is http only, secure,
// and set to same site strict mode.
func (j *Auth) GetRefreshCookie(refreshToken string) *http.Cookie {
	return &http.Cookie{
		Name:     j.CookieName,
		Path:     j.CookiePath,
		Value:    refreshToken,
		Expires:  time.Now().Add(j.RefreshExpiry),
		MaxAge:   int(j.RefreshExpiry.Seconds()),
		SameSite: http.SameSiteStrictMode,
		Domain:   j.CookieDomain,
		HttpOnly: true,
		Secure:   true,
	}
}

// GetExpiredRefreshCookie is a convenience method to return a cookie suitable for forcing a user's browser
// to delete the existing cookie.
func (j *Auth) GetExpiredRefreshCookie() *http.Cookie {
	return &http.Cookie{
		Name:     j.CookieName,
		Value:    "",
		Domain:   j.CookieDomain,
		Path:     j.CookiePath,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	}
}
