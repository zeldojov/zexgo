package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zeldojov/zexgo/internal/session"
	"github.com/zeldojov/zexgo/internal/store"
)

func publicChain(st *store.Store, handler http.Handler) http.Handler {
	return AllowedMethods(
		Session(
			st,
			ValidateSession(
				st,
				CSRF(handler),
			),
		),
	)
}

func TestCSRF_GET(t *testing.T) {
	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestCSRF_POSTWithoutSession(t *testing.T) {
	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestCSRF_POSTWithValidCSRFToken(t *testing.T) {
	baseReq := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := newAnonymousSession(t, baseReq)

	form := url.Values{}
	form.Set(session.CSRFFieldName(), sess.CSRFToken())

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestCSRF_POSTWithInvalidCSRFToken(t *testing.T) {
	baseReq := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := newAnonymousSession(t, baseReq)

	form := url.Values{}
	form.Set(session.CSRFFieldName(), "invalid-token")

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestCSRF_POSTWithoutCSRFToken(t *testing.T) {
	baseReq := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := newAnonymousSession(t, baseReq)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		nil,
	)

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestPublicChain_GET(t *testing.T) {
	st := setupTestStore(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	publicChain(st, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestPublicChain_POSTWithoutSession(t *testing.T) {
	st := setupTestStore(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	publicChain(st, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestPublicChain_POSTWithValidCSRF(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := newAnonymousSession(t, req)

	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), sess.CSRFToken())

	req = httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	addSessionCookie(t, req, sess.ID())

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	publicChain(st, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestPublicChain_POSTWithInvalidCSRF(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := newAnonymousSession(t, req)

	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), "invalid-token")

	req = httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	addSessionCookie(t, req, sess.ID())

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	publicChain(st, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}
