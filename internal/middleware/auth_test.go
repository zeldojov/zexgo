package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"uuid"

	"github.com/zeldojov/zexgo/internal/session"
	"github.com/zeldojov/zexgo/internal/store"
	"github.com/zeldojov/zexgo/internal/user"
	"github.com/zeldojov/zexgo/internal/zpass"
)

func newAnonymousSession(t *testing.T, req *http.Request) *session.Session {
	t.Helper()

	sess, err := session.NewSessionFromReq(req)
	if err != nil {
		t.Fatalf("create anonymous session: %v", err)
	}

	return sess
}

func newAuthenticatedSession(
	t *testing.T,
	userID uuid.UUID,
	req *http.Request,
) *session.Session {
	t.Helper()

	data, err := session.GetSessionDataFromReq(req)
	if err != nil {
		t.Fatalf("create authenticated session data: %v", err)
	}

	data.UserID = &userID
	return session.NewSessionFromData(data)
}

func createTestUser(t *testing.T, st *store.Store) uuid.UUID {
	t.Helper()

	testUser := user.CreateUser(
		"1234567890123",
		"testuser",
		"Test User",
		"test@example.com",
		zpass.PasswordHash("test-password-hash"),
	)

	if err := st.CreateUser(context.Background(), testUser); err != nil {
		t.Fatalf("create test user: %v", err)
	}

	return testUser.ID()
}

func addSessionCookie(t *testing.T, req *http.Request, sessionID string) {
	t.Helper()

	rec := httptest.NewRecorder()
	if err := session.SetSessionCookie(rec, sessionID); err != nil {
		t.Fatalf("set session cookie: %v", err)
	}

	req.AddCookie(rec.Result().Cookies()[0])
}

func authChain(st *store.Store, handler http.Handler) http.Handler {
	return AllowedMethods(
		Session(
			st,
			ValidateSession(
				st,
				CSRF(
					Auth()(handler),
				),
			),
		),
	)
}

func TestAuth_NoSession(t *testing.T) {
	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Auth()(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuth_AnonymousSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := newAnonymousSession(t, req)

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	Auth()(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rec.Code)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("expected redirect to /login, got %q", location)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuth_AuthenticatedSession(t *testing.T) {
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := newAuthenticatedSession(t, userID, req)

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	Auth()(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestAuthChain_GETAnonymous(t *testing.T) {
	st := setupTestStore(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)
	rec := httptest.NewRecorder()

	authChain(st, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rec.Code)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("expected redirect to /login, got %q", location)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuthChain_POSTWithoutSession(t *testing.T) {
	st := setupTestStore(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodPost, "/user/home", nil)
	rec := httptest.NewRecorder()

	authChain(st, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuthChain_AuthenticatedGET(t *testing.T) {
	st := setupTestStore(t)

	userID := createTestUser(t, st)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sess := newAuthenticatedSession(t, userID, req)

	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/user/home", nil)
	addSessionCookie(t, req, sess.ID())

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	authChain(st, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestAuthChain_AuthenticatedPOST(t *testing.T) {
	st := setupTestStore(t)

	userID := createTestUser(t, st)

	req := httptest.NewRequest(http.MethodPost, "/", nil)

	sess := newAuthenticatedSession(t, userID, req)

	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), sess.CSRFToken())

	req = httptest.NewRequest(
		http.MethodPost,
		"/user/home",
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

	authChain(st, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestAuthChain_InvalidCSRF(t *testing.T) {
	st := setupTestStore(t)

	userID := createTestUser(t, st)

	req := httptest.NewRequest(http.MethodPost, "/", nil)

	sess := newAuthenticatedSession(t, userID, req)

	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), "invalid-token")

	req = httptest.NewRequest(
		http.MethodPost,
		"/user/home",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	addSessionCookie(t, req, sess.ID())

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	authChain(st, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}
