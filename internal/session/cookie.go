package session

import (
	"errors"
	"net/http"
)

var (
	// ErrCookieNotFound indicates that the requested cookie could not be read.
	ErrCookieNotFound = errors.New("cookie not found")
	// ErrCookieEmpty indicates that a cookie was found without a value.
	ErrCookieEmpty = errors.New("cookie is empty")
	// ErrCookieInvalid indicates that a cookie value has an unexpected length.
	ErrCookieInvalid = errors.New("cookie value is invalid")

	sessionCookie = cookieConfig{
		Name:     "__Host-session_id",
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	rememberCookie = cookieConfig{
		Name:     "__Host-remember_token",
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60,
	}
)

const cookieTokenLength = 64

type cookieConfig struct {
	Name     string
	Path     string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
	MaxAge   int
}

// region API

// GetSessionCookie returns the session cookie from the request.
//
// It returns ErrCookieNotFound when http.Request.Cookie reports an error,
// ErrCookieEmpty when the cookie has no value, or ErrCookieInvalid when the
// value is not cookieTokenLength characters long.
func GetSessionCookie(r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie(sessionCookie.Name)
	if err != nil {
		return nil, ErrCookieNotFound
	}

	if cookie.Value == "" {
		return nil, ErrCookieEmpty
	}
	if len(cookie.Value) != cookieTokenLength {
		return nil, ErrCookieInvalid
	}

	return cookie, nil
}

// GetRememberCookie returns the remember-me cookie from the request.
//
// It returns ErrCookieNotFound when http.Request.Cookie reports an error,
// ErrCookieEmpty when the cookie has no value, or ErrCookieInvalid when the
// value is not cookieTokenLength characters long.
func GetRememberCookie(r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie(rememberCookie.Name)
	if err != nil {
		return nil, ErrCookieNotFound
	}

	if cookie.Value == "" {
		return nil, ErrCookieEmpty
	}
	if len(cookie.Value) != cookieTokenLength {
		return nil, ErrCookieInvalid
	}

	return cookie, nil
}

// SetSessionCookie writes a session cookie containing sessionID.
//
// It returns ErrCookieEmpty for an empty sessionID or ErrCookieInvalid when
// sessionID is not cookieTokenLength characters long.
func SetSessionCookie(w http.ResponseWriter, sessionID string) error {
	if sessionID == "" {
		return ErrCookieEmpty
	}
	if len(sessionID) != cookieTokenLength {
		return ErrCookieInvalid
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie.Name,
		Value:    sessionID,
		Path:     sessionCookie.Path,
		Secure:   sessionCookie.Secure,
		HttpOnly: sessionCookie.HTTPOnly,
		SameSite: sessionCookie.SameSite,
		MaxAge:   sessionCookie.MaxAge,
	})

	return nil
}

// SetRememberCookie writes a persistent remember-me cookie containing token.
//
// It returns ErrCookieEmpty for an empty token or ErrCookieInvalid when token
// is not cookieTokenLength characters long.
func SetRememberCookie(w http.ResponseWriter, token string) error {
	if token == "" {
		return ErrCookieEmpty
	}
	if len(token) != cookieTokenLength {
		return ErrCookieInvalid
	}

	http.SetCookie(w, &http.Cookie{
		Name:     rememberCookie.Name,
		Value:    token,
		Path:     rememberCookie.Path,
		Secure:   rememberCookie.Secure,
		HttpOnly: rememberCookie.HTTPOnly,
		SameSite: rememberCookie.SameSite,
		MaxAge:   rememberCookie.MaxAge,
	})

	return nil
}

// UnsetSessionCookie expires the session cookie in the response.
func UnsetSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie.Name,
		Value:    "",
		Path:     sessionCookie.Path,
		MaxAge:   -1,
		Secure:   sessionCookie.Secure,
		HttpOnly: sessionCookie.HTTPOnly,
		SameSite: sessionCookie.SameSite,
	})
}

// UnsetRememberCookie expires the remember-me cookie in the response.
func UnsetRememberCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     rememberCookie.Name,
		Value:    "",
		Path:     rememberCookie.Path,
		MaxAge:   -1,
		Secure:   rememberCookie.Secure,
		HttpOnly: rememberCookie.HTTPOnly,
		SameSite: rememberCookie.SameSite,
	})
}

// endregion API
