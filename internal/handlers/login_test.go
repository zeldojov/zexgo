package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zeldojov/zexgo/internal/session"
	"github.com/zeldojov/zexgo/internal/user"
	"github.com/zeldojov/zexgo/internal/zpass"
)

func addSessionCookieForTest(t *testing.T, req *http.Request, id string) {
	t.Helper()

	rec := httptest.NewRecorder()
	if err := session.SetSessionCookie(rec, id); err != nil {
		t.Fatal(err)
	}

	req.AddCookie(rec.Result().Cookies()[0])
}

func TestLogin_UserNotFound(t *testing.T) {
	st := setupTestStore(t)
	handler := newPageTestHandler(t, st)

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(url.Values{
			"username": {"nonexistent"},
			"password": {"ValidPassword123!"},
		}.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	sess := newTestSession(t, req)

	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}

	addSessionCookieForTest(t, req, sess.ID())

	ctx := context.WithValue(
		req.Context(),
		session.ContextKey{},
		sess,
	)

	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf(
			"expected redirect to /login, got %q",
			location,
		)
	}

	if message, ok := sess.GetFlash("error"); !ok {
		t.Fatal("expected flash error")
	} else if message != "invalid credentials" {
		t.Fatalf(
			"expected flash error %q, got %q",
			"invalid credentials",
			message,
		)
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	st := setupTestStore(t)
	handler := newPageTestHandler(t, st)

	pass := "ValidPassword123!"

	passwordHash, err := zpass.NewPasswordHash(pass)
	if err != nil {
		t.Fatal(err)
	}

	newUser := user.CreateUser(
		"1234567890123",
		"testuser",
		"Test User",
		"test@example.com",
		passwordHash,
	)

	if err := st.CreateUser(context.Background(), newUser); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(url.Values{
			"username": {"testuser"},
			"password": {"WrongPassword123!"},
		}.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	sess := newTestSession(t, req)

	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}

	addSessionCookieForTest(t, req, sess.ID())

	ctx := context.WithValue(
		req.Context(),
		session.ContextKey{},
		sess,
	)

	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf(
			"expected redirect to /login, got %q",
			location,
		)
	}

	if message, ok := sess.GetFlash("error"); !ok {
		t.Fatal("expected flash error")
	} else if message != "invalid credentials" {
		t.Fatalf(
			"expected flash error %q, got %q",
			"invalid credentials",
			message,
		)
	}
}

func TestLogin_Success(t *testing.T) {
	st := setupTestStore(t)
	handler := newPageTestHandler(t, st)

	pass := "ValidPassword123!"

	passwordHash, err := zpass.NewPasswordHash(pass)
	if err != nil {
		t.Fatal(err)
	}

	newUser := user.CreateUser(
		"1234567890123",
		"testuser",
		"Test User",
		"test@example.com",
		passwordHash,
	)

	if err := st.CreateUser(context.Background(), newUser); err != nil {
		t.Fatal(err)
	}

	initialReq := httptest.NewRequest(
		http.MethodGet,
		"/login",
		nil,
	)

	sess := newTestSession(t, initialReq)

	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}

	oldSessionID := sess.ID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(url.Values{
			"username": {"testuser"},
			"password": {pass},
		}.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	addSessionCookieForTest(t, req, oldSessionID)

	ctx := context.WithValue(
		req.Context(),
		session.ContextKey{},
		sess,
	)

	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/user/home" {
		t.Fatalf(
			"expected redirect to /user/home, got %q",
			location,
		)
	}

	if sess.ID() == oldSessionID {
		t.Fatal("expected session ID to change")
	}

	if !sess.IsAuthenticated() {
		t.Fatal("expected session to be authenticated")
	}

	if sess.UserID() == nil {
		t.Fatal("expected user ID in session")
	}

	if *sess.UserID() != newUser.ID() {
		t.Fatalf(
			"expected session user ID %q, got %q",
			newUser.ID(),
			*sess.UserID(),
		)
	}

	_, err = st.GetSessionDataFromDB(context.Background(), oldSessionID)

	if err == nil {
		t.Fatal("expected old session to be deleted")
	}

	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf(
			"expected ErrSessionNotFound for old session, got %v",
			err,
		)
	}

	savedSession := loadTestSession(t, st, sess.ID())

	if !savedSession.IsAuthenticated() {
		t.Fatal("expected saved session to be authenticated")
	}

	if savedSession.UserID() == nil {
		t.Fatal("expected saved session user ID")
	}

	if *savedSession.UserID() != newUser.ID() {
		t.Fatalf(
			"expected saved session user ID %q, got %q",
			newUser.ID(),
			*savedSession.UserID(),
		)
	}

	setCookies := rec.Result().Cookies()

	if len(setCookies) == 0 {
		t.Fatal("expected Set-Cookie header")
	}

	var foundNewSessionCookie bool

	for _, cookie := range setCookies {
		if cookie.Value == sess.ID() {
			foundNewSessionCookie = true
			break
		}
	}

	if !foundNewSessionCookie {
		t.Fatalf("expected Set-Cookie for session %q", sess.ID())
	}
}
