package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func validCookieToken() string {
	return strings.Repeat("a", cookieTokenLength)
}

func TestGetCookie(t *testing.T) {
	tests := []struct {
		name     string
		cookie   *http.Cookie
		get      func(*http.Request) (*http.Cookie, error)
		wantErr  error
		wantName string
	}{
		{
			name:    "session cookie not found",
			get:     GetSessionCookie,
			wantErr: ErrCookieNotFound,
		},
		{
			name:    "session cookie empty",
			cookie:  &http.Cookie{Name: sessionCookie.Name},
			get:     GetSessionCookie,
			wantErr: ErrCookieEmpty,
		},
		{
			name:    "session cookie invalid length",
			cookie:  &http.Cookie{Name: sessionCookie.Name, Value: "short"},
			get:     GetSessionCookie,
			wantErr: ErrCookieInvalid,
		},
		{
			name:     "session cookie valid",
			cookie:   &http.Cookie{Name: sessionCookie.Name, Value: validCookieToken()},
			get:      GetSessionCookie,
			wantName: sessionCookie.Name,
		},
		{
			name:    "remember cookie not found",
			get:     GetRememberCookie,
			wantErr: ErrCookieNotFound,
		},
		{
			name:    "remember cookie empty",
			cookie:  &http.Cookie{Name: rememberCookie.Name},
			get:     GetRememberCookie,
			wantErr: ErrCookieEmpty,
		},
		{
			name:    "remember cookie invalid length",
			cookie:  &http.Cookie{Name: rememberCookie.Name, Value: "short"},
			get:     GetRememberCookie,
			wantErr: ErrCookieInvalid,
		},
		{
			name:     "remember cookie valid",
			cookie:   &http.Cookie{Name: rememberCookie.Name, Value: validCookieToken()},
			get:      GetRememberCookie,
			wantName: rememberCookie.Name,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.cookie != nil {
				request.AddCookie(test.cookie)
			}

			cookie, err := test.get(request)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				if cookie != nil {
					t.Fatalf("cookie = %#v, want nil", cookie)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cookie == nil || cookie.Name != test.wantName || cookie.Value != validCookieToken() {
				t.Fatalf("cookie = %#v, want name %q and valid token", cookie, test.wantName)
			}
		})
	}
}

func TestSetCookie(t *testing.T) {
	tests := []struct {
		name       string
		set        func(http.ResponseWriter, string) error
		config     cookieConfig
		wantErr    error
		wantCookie string
	}{
		{
			name:    "session empty",
			set:     SetSessionCookie,
			config:  sessionCookie,
			wantErr: ErrCookieEmpty,
		},
		{
			name:    "session invalid length",
			set:     SetSessionCookie,
			config:  sessionCookie,
			wantErr: ErrCookieInvalid,
		},
		{
			name:       "session valid",
			set:        SetSessionCookie,
			config:     sessionCookie,
			wantCookie: sessionCookie.Name,
		},
		{
			name:    "remember empty",
			set:     SetRememberCookie,
			config:  rememberCookie,
			wantErr: ErrCookieEmpty,
		},
		{
			name:    "remember invalid length",
			set:     SetRememberCookie,
			config:  rememberCookie,
			wantErr: ErrCookieInvalid,
		},
		{
			name:       "remember valid",
			set:        SetRememberCookie,
			config:     rememberCookie,
			wantCookie: rememberCookie.Name,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validCookieToken()
			if test.wantErr == ErrCookieEmpty {
				value = ""
			}
			if test.wantErr == ErrCookieInvalid {
				value = "short"
			}

			recorder := httptest.NewRecorder()
			err := test.set(recorder, value)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				if recorder.Header().Get("Set-Cookie") != "" {
					t.Fatal("Set-Cookie header was written for invalid value")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cookies := recorder.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("cookies = %d, want 1", len(cookies))
			}

			cookie := cookies[0]
			if cookie.Name != test.wantCookie || cookie.Value != value {
				t.Fatalf("cookie = %#v, want %q=%q", cookie, test.wantCookie, value)
			}
			if cookie.Path != test.config.Path || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != test.config.SameSite || cookie.MaxAge != test.config.MaxAge {
				t.Fatalf("cookie attributes = %#v, want config %#v", cookie, test.config)
			}
		})
	}
}

func TestUnsetCookie(t *testing.T) {
	tests := []struct {
		name   string
		unset  func(http.ResponseWriter)
		config cookieConfig
	}{
		{name: "session", unset: UnsetSessionCookie, config: sessionCookie},
		{name: "remember", unset: UnsetRememberCookie, config: rememberCookie},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.unset(recorder)

			cookies := recorder.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("cookies = %d, want 1", len(cookies))
			}

			cookie := cookies[0]
			if cookie.Name != test.config.Name || cookie.Value != "" || cookie.Path != test.config.Path || cookie.MaxAge != -1 {
				t.Fatalf("cookie = %#v, want deleted %s cookie", cookie, test.config.Name)
			}
		})
	}
}
