package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"uuid"

	"github.com/zeldojov/zexgo/internal/session"
)

func middlewareTestApp() *application {
	return &application{views: middlewareTestRenderer{}}
}

func newAuthenticatedAppSession(t *testing.T, req *http.Request) *session.Session {
	t.Helper()

	userID := uuid.New()
	data, err := session.GetSessionDataFromReq(req)
	if err != nil {
		t.Fatal(err)
	}
	data.UserID = &userID
	return session.NewSessionFromData(data)
}

func requestWithSession(t *testing.T, method string, sess *session.Session) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, "/", nil)
	return session.SetSession(sess, req)
}

func TestAppAuthMissingSession(t *testing.T) {
	called := false
	handler := middlewareTestApp().Auth(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError || called {
		t.Fatalf("expected 500 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestAppAuthAnonymousGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = session.SetSession(newAnonymousAppSession(t, req), req)
	called := false
	handler := middlewareTestApp().Auth(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" || called {
		t.Fatalf("expected redirect to /login without calling next, got %d, %q, %v", rec.Code, rec.Header().Get("Location"), called)
	}
}

func TestAppAuthAnonymousPOST(t *testing.T) {
	req := requestWithSession(t, http.MethodPost, newAnonymousAppSession(t, httptest.NewRequest(http.MethodPost, "/", nil)))
	called := false
	handler := middlewareTestApp().Auth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized || called {
		t.Fatalf("expected 401 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestAppAuthAuthenticated(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = session.SetSession(newAuthenticatedAppSession(t, req), req)
	called := false
	handler := middlewareTestApp().Auth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(http.StatusOK) }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected authenticated request to continue, got %d and %v", rec.Code, called)
	}
}

func TestAppGuestMissingSession(t *testing.T) {
	called := false
	handler := middlewareTestApp().Guest(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError || called {
		t.Fatalf("expected 500 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestAppGuestAuthenticated(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = session.SetSession(newAuthenticatedAppSession(t, req), req)
	called := false
	handler := middlewareTestApp().Guest(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/user/home" || called {
		t.Fatalf("expected redirect to /user/home without calling next, got %d, %q, %v", rec.Code, rec.Header().Get("Location"), called)
	}
}

func TestAppGuestAnonymous(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = session.SetSession(newAnonymousAppSession(t, req), req)
	called := false
	handler := middlewareTestApp().Guest(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(http.StatusOK) }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected anonymous request to continue, got %d and %v", rec.Code, called)
	}
}

func TestAppCSRFGET(t *testing.T) {
	called := false
	handler := middlewareTestApp().CSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(http.StatusOK) }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected GET to continue, got %d and %v", rec.Code, called)
	}
}

func TestAppCSRFPostMissingSession(t *testing.T) {
	called := false
	handler := middlewareTestApp().CSRF(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	if rec.Code != http.StatusInternalServerError || called {
		t.Fatalf("expected 500 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestAppCSRFPostToken(t *testing.T) {
	baseReq := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := newAnonymousAppSession(t, baseReq)
	form := url.Values{session.CSRFFieldName(): {sess.CSRFToken()}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.SetSession(sess, req)
	called := false
	handler := middlewareTestApp().CSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(http.StatusOK) }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected valid token to continue, got %d and %v", rec.Code, called)
	}
}

func TestAppCSRFPostInvalidOrMissingToken(t *testing.T) {
	for _, token := range []string{"invalid-token", ""} {
		t.Run(token, func(t *testing.T) {
			baseReq := httptest.NewRequest(http.MethodPost, "/", nil)
			sess := newAnonymousAppSession(t, baseReq)
			form := url.Values{}
			if token != "" {
				form.Set(session.CSRFFieldName(), token)
			}
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
			req = session.SetSession(sess, req)
			called := false
			handler := middlewareTestApp().CSRF(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden || called {
				t.Fatalf("expected 403 without calling next, got %d and %v", rec.Code, called)
			}
		})
	}
}
