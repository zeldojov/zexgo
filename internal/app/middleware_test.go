package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zeldojov/zexgo/internal/ratelimiter"
	"github.com/zeldojov/zexgo/internal/session"
)

type middlewareTestRenderer struct{}

func (middlewareTestRenderer) ExecuteTemplate(io.Writer, string, any) error {
	return nil
}

func TestAllowedMethods(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		nextCalled     bool
	}{
		{
			name:           "GET allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			nextCalled:     true,
		},
		{
			name:           "POST allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			nextCalled:     true,
		},
		{
			name:           "PUT not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			nextCalled:     false,
		},
		{
			name:           "DELETE not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			nextCalled:     false,
		},
		{
			name:           "PATCH not allowed",
			method:         http.MethodPatch,
			expectedStatus: http.StatusMethodNotAllowed,
			nextCalled:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			app := &application{views: middlewareTestRenderer{}}
			handler := app.AllowedMethods(next)
			req := httptest.NewRequest(tt.method, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			if nextCalled != tt.nextCalled {
				t.Fatalf("expected nextCalled=%v, got %v", tt.nextCalled, nextCalled)
			}
			if !tt.nextCalled && rec.Header().Get("Allow") != "GET, HEAD, POST" {
				t.Fatalf("expected Allow header, got %q", rec.Header().Get("Allow"))
			}
		})
	}
}

func TestCreateChainPreservesMiddlewareOrder(t *testing.T) {
	app := &application{}
	var order []string
	wrap := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-before")
				next.ServeHTTP(w, r)
				order = append(order, name+"-after")
			})
		}
	}

	handler := app.CreateChain(wrap("first"), wrap("second"))(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			order = append(order, "handler")
		}),
	)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	expected := []string{"first-before", "second-before", "handler", "second-after", "first-after"}
	if !slices.Equal(order, expected) {
		t.Fatalf("expected middleware order %v, got %v", expected, order)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := ratelimiter.New(1, time.Minute)
	called := 0
	handler := (&application{}).RateLimiter(limiter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))

	if first.Code != http.StatusOK {
		t.Fatalf("expected first request status 200, got %d", first.Code)
	}
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request status 429, got %d", second.Code)
	}
	if called != 1 {
		t.Fatalf("expected next handler to be called once, got %d", called)
	}
}

func appSessionCookie(t *testing.T, id string) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	if err := session.SetSessionCookie(rec, id); err != nil {
		t.Fatal(err)
	}
	return rec.Result().Cookies()[0]
}

func newAnonymousAppSession(t *testing.T, req *http.Request) *session.Session {
	return newTestSession(t, req)
}

func TestSession_GETCreatesSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	called := false
	handler := app.Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if sess, ok := session.GetSession(r); !ok || sess == nil {
			t.Fatal("expected session in context")
		}
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected 200 and next handler call, got %d and %v", rec.Code, called)
	}
	if len(rec.Result().Cookies()) == 0 || rec.Result().Cookies()[0].Value == "" {
		t.Fatal("expected session cookie")
	}
}

func TestSession_POSTWithoutSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	called := false
	handler := app.Session(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("expected 403 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestSession_GETRecreatesMissingSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	oldCookie := appSessionCookie(t, strings.Repeat("x", 64))
	var loadedID string
	handler := app.Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loaded, ok := session.GetSession(r)
		if !ok {
			t.Fatal("expected session in context")
		}
		loadedID = loaded.ID()
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(oldCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || loadedID == oldCookie.Value {
		t.Fatalf("expected new session, got status %d and id %q", rec.Code, loadedID)
	}
}

func TestSession_LoadsExistingSessionGETAndPOST(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			st := setupTestStore(t)
			app := newTestApplication(t, st)
			createReq := httptest.NewRequest(http.MethodGet, "/", nil)
			sess := newAnonymousAppSession(t, createReq)
			if err := st.SaveSession(createReq.Context(), sess); err != nil {
				t.Fatal(err)
			}
			cookie := appSessionCookie(t, sess.ID())
			called := false
			handler := app.Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				loaded, ok := session.GetSession(r)
				if !ok || loaded.ID() != sess.ID() {
					t.Fatalf("expected session %q in context", sess.ID())
				}
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(method, "/", nil)
			req.AddCookie(cookie)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK || !called {
				t.Fatalf("expected existing session request to continue, got %d and %v", rec.Code, called)
			}
		})
	}
}

func TestSession_POSTRejectsMissingDatabaseSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	cookie := appSessionCookie(t, strings.Repeat("x", 64))
	called := false
	handler := app.Session(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("expected 403 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestValidateSession_NoSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	called := false
	handler := app.ValidateSession(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError || called {
		t.Fatalf("expected 500 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestValidateSession_RecreatesExpiredGET(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := newAnonymousAppSession(t, req)
	sess.SetExpiresAt(time.Now().Add(-time.Minute))
	if err := st.SaveSession(req.Context(), sess); err != nil {
		t.Fatal(err)
	}
	oldID := sess.ID()
	req = session.SetSession(sess, req)
	var loadedID string
	handler := app.ValidateSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loaded, _ := session.GetSession(r)
		loadedID = loaded.ID()
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || loadedID == oldID {
		t.Fatalf("expected recreated session, got status %d and id %q", rec.Code, loadedID)
	}
}

func TestValidateSession_RejectsExpiredPOST(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := newAnonymousAppSession(t, req)
	sess.SetExpiresAt(time.Now().Add(-time.Minute))
	req = session.SetSession(sess, req)
	called := false
	handler := app.ValidateSession(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("expected 403 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestValidateSession_ValidGETAndPOST(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			st := setupTestStore(t)
			app := newTestApplication(t, st)
			req := httptest.NewRequest(method, "/", nil)
			sess := newAnonymousAppSession(t, req)
			if err := st.SaveSession(req.Context(), sess); err != nil {
				t.Fatal(err)
			}
			req = session.SetSession(sess, req)
			called := false
			handler := app.ValidateSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK || !called {
				t.Fatalf("expected valid request to continue, got %d and %v", rec.Code, called)
			}
		})
	}
}

func TestValidateSession_RecreatesIPMismatchGET(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := newAnonymousAppSession(t, req)
	sess.SetUserIP("1.2.3.4")
	if err := st.SaveSession(req.Context(), sess); err != nil {
		t.Fatal(err)
	}
	oldID := sess.ID()
	req = session.SetSession(sess, req)
	var loadedID string
	handler := app.ValidateSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loaded, _ := session.GetSession(r)
		loadedID = loaded.ID()
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || loadedID == oldID {
		t.Fatalf("expected IP mismatch recreation, got %d and %q", rec.Code, loadedID)
	}
}

func TestValidateSession_RejectsIPMismatchPOST(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := newAnonymousAppSession(t, req)
	sess.SetUserIP("1.2.3.4")
	req = session.SetSession(sess, req)
	called := false
	handler := app.ValidateSession(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("expected 403 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestValidateSession_RecreatesUserAgentMismatchGET(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "test-agent-1")
	sess := newAnonymousAppSession(t, req)
	sess.SetUserAgent("test-agent-2")
	if err := st.SaveSession(req.Context(), sess); err != nil {
		t.Fatal(err)
	}
	oldID := sess.ID()
	req = session.SetSession(sess, req)
	var loadedID string
	handler := app.ValidateSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loaded, _ := session.GetSession(r)
		loadedID = loaded.ID()
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || loadedID == oldID {
		t.Fatalf("expected user-agent mismatch recreation, got %d and %q", rec.Code, loadedID)
	}
}

func TestValidateSession_RejectsUserAgentMismatchPOST(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("User-Agent", "test-agent-1")
	sess := newAnonymousAppSession(t, req)
	sess.SetUserAgent("test-agent-2")
	req = session.SetSession(sess, req)
	called := false
	handler := app.ValidateSession(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("expected 403 without calling next, got %d and %v", rec.Code, called)
	}
}

func TestValidateSession_RefreshesSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := newAnonymousAppSession(t, req)
	oldExpiration := time.Now().Add(session.SessionDuration() / 4)
	sess.SetExpiresAt(oldExpiration)
	req = session.SetSession(sess, req)
	handler := app.ValidateSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !sess.ExpiresAt().After(oldExpiration) {
		t.Fatalf("expected refreshed session, got status %d and expiration %v", rec.Code, sess.ExpiresAt())
	}
}

func TestValidateSession_RecreatesSessionCookie(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := newAnonymousAppSession(t, req)
	sess.SetExpiresAt(time.Now().Add(-time.Minute))
	oldID := sess.ID()
	req = session.SetSession(sess, req)
	handler := app.ValidateSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Value != "" && cookie.Value != oldID {
			return
		}
	}
	t.Fatal("expected new session cookie")
}
