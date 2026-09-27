package app

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"uuid"

	"github.com/zeldojov/zexgo/internal/session"
	"github.com/zeldojov/zexgo/internal/store"
	"github.com/zeldojov/zexgo/internal/user"
	viewspkg "github.com/zeldojov/zexgo/internal/views"
	"github.com/zeldojov/zexgo/internal/zpass"
)

func setupTestStore(t *testing.T) *store.Store {
	t.Helper()

	db, err := store.Connect(
		"root",
		"root",
		"127.0.0.1",
		"5432",
		"gozex_test",
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	st, err := store.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("DELETE FROM sessions"); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("DELETE FROM users"); err != nil {
		t.Fatal(err)
	}

	return st
}

func newTestApplication(t *testing.T, st *store.Store) *application {
	t.Helper()

	views, err := viewspkg.New(os.DirFS("../.."), "templates", template.FuncMap{
		"upper": strings.ToUpper,
	})
	if err != nil {
		t.Fatal(err)
	}

	return &application{
		store: st,
		views: views,
	}
}

func newTestSession(t *testing.T, req *http.Request) *session.Session {
	t.Helper()

	sess, err := session.NewSessionFromReq(req)
	if err != nil {
		t.Fatal(err)
	}

	return sess
}

func loadTestSession(t *testing.T, st *store.Store, id string) *session.Session {
	t.Helper()

	data, err := st.GetSessionDataFromDB(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}

	return session.NewSessionFromData(data)
}

func parseSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	if cookies := rec.Result().Cookies(); len(cookies) > 0 {
		return cookies[0]
	}

	t.Fatal("session cookie not found")
	return nil
}

func addSessionCookieForTest(t *testing.T, req *http.Request, id string) {
	t.Helper()

	rec := httptest.NewRecorder()
	if err := session.SetSessionCookie(rec, id); err != nil {
		t.Fatal(err)
	}

	req.AddCookie(rec.Result().Cookies()[0])
}

func publicChain(st *store.Store, handler http.Handler) http.Handler {
	app := &application{store: st, views: middlewareTestRenderer{}}
	return app.CreateChain(app.Session, app.ValidateSession, app.CSRF)(handler)
}

func TestRegisterPage(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	sess := newTestSession(t, req)
	req = session.SetSession(sess, req)

	rec := httptest.NewRecorder()
	app.RegisterPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `action="/register"`) {
		t.Fatal(`expected form action "/register"`)
	}
	if !strings.Contains(body, `method="POST"`) {
		t.Fatal(`expected form method "POST"`)
	}

	expectedCSRF := `<input type="hidden" name="csrf_token" value="` + sess.CSRFToken() + `">`
	if !strings.Contains(body, expectedCSRF) {
		t.Fatal("expected session CSRF token in form")
	}
}

func TestRegisterPage_MissingSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rec := httptest.NewRecorder()

	app.RegisterPage(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestRegisterPage_ShowsFlashError(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	sess := newTestSession(t, req)
	sess.SetFlash("error", "invalid username")
	req = session.SetSession(sess, req)

	rec := httptest.NewRecorder()
	app.RegisterPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid username") {
		t.Fatal(`expected body to contain "invalid username"`)
	}
	if sess.HasValue("flash:error") {
		t.Fatal("expected flash error to be consumed")
	}
}

func TestRegisterFlashFlow(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	registerHandler := publicChain(st, http.HandlerFunc(app.Register))
	registerPageHandler := publicChain(st, http.HandlerFunc(app.RegisterPage))

	getReq := httptest.NewRequest(http.MethodGet, "/register", nil)
	getRec := httptest.NewRecorder()
	registerPageHandler.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET /register status %d, got %d", http.StatusOK, getRec.Code)
	}

	cookie := parseSessionCookie(t, getRec)
	sess := loadTestSession(t, st, cookie.Value)
	form := url.Values{
		"username":   {"ab"},
		"password":   {"ValidPassword123!"},
		"csrf_token": {sess.CSRFToken()},
	}

	postReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(cookie)
	postRec := httptest.NewRecorder()
	registerHandler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("expected POST /register status %d, got %d", http.StatusSeeOther, postRec.Code)
	}
	if location := postRec.Header().Get("Location"); location != "/register" {
		t.Fatalf("expected redirect to /register, got %q", location)
	}

	savedSession := loadTestSession(t, st, cookie.Value)
	if !savedSession.HasValue("flash:error") {
		t.Fatal("expected flash error to be saved")
	}

	finalReq := httptest.NewRequest(http.MethodGet, "/register", nil)
	finalReq.AddCookie(cookie)
	finalRec := httptest.NewRecorder()
	registerPageHandler.ServeHTTP(finalRec, finalReq)

	if finalRec.Code != http.StatusOK {
		t.Fatalf("expected GET /register status %d, got %d", http.StatusOK, finalRec.Code)
	}
	if !strings.Contains(finalRec.Body.String(), "invalid username") {
		t.Fatal(`expected page to contain "invalid username"`)
	}

	savedSession = loadTestSession(t, st, cookie.Value)
	if savedSession.HasValue("flash:error") {
		t.Fatal("expected flash error to be consumed")
	}
}

func TestRegister_InvalidUsername(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(url.Values{
		"username": {"ab"},
		"password": {"ValidPassword123!"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sess := newTestSession(t, req)
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()

	app.Register(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/register" {
		t.Fatalf("expected redirect to /register, got %q", location)
	}
	if message, ok := sess.GetFlash("error"); !ok || message != "invalid username" {
		t.Fatalf("expected flash error %q, got %q", "invalid username", message)
	}
}

func TestRegister_InvalidPassword(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(url.Values{
		"username": {"testuser"},
		"password": {"123"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sess := newTestSession(t, req)
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()

	app.Register(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/register" {
		t.Fatalf("expected redirect to /register, got %q", location)
	}
	if message, ok := sess.GetFlash("error"); !ok || message != "invalid password" {
		t.Fatalf("expected flash error %q, got %q", "invalid password", message)
	}
}

func TestRegister_UsernameTaken(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	existingUser := user.CreateUser("1234567890123", "testuser", "Existing User", "existing@example.com", "existing-hash")
	if err := st.CreateUser(context.Background(), existingUser); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(url.Values{
		"username": {"testuser"},
		"password": {"ValidPassword123!"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sess := newTestSession(t, req)
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()

	app.Register(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/register" {
		t.Fatalf("expected redirect to /register, got %q", location)
	}
	if message, ok := sess.GetFlash("error"); !ok || message != "username already exists" {
		t.Fatalf("expected flash error %q, got %q", "username already exists", message)
	}
}

func TestRegister_Success(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	initialReq := httptest.NewRequest(http.MethodGet, "/register", nil)
	sess := newTestSession(t, initialReq)
	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	oldSessionID := sess.ID()

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(url.Values{
		"username":  {"testuser"},
		"password":  {"ValidPassword123!"},
		"jmbg":      {"1234567890123"},
		"full_name": {"Test User"},
		"email":     {"test@example.com"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addSessionCookieForTest(t, req, oldSessionID)
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()

	app.Register(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/user/home" {
		t.Fatalf("expected redirect to /user/home, got %q", location)
	}

	foundUser, err := st.GetUserByUsername(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("failed to get registered user: %v", err)
	}
	if foundUser.Username() != "testuser" || foundUser.JMBG() != "1234567890123" ||
		foundUser.FullName() != "Test User" || foundUser.Email() != "test@example.com" {
		t.Fatal("registered user fields do not match")
	}
	if foundUser.PasswordHash() == "ValidPassword123!" {
		t.Fatal("password was stored in plaintext")
	}
	if sess.ID() == oldSessionID || !sess.IsAuthenticated() || sess.UserID() == nil {
		t.Fatal("expected a new authenticated session")
	}

	savedData, err := st.GetSessionDataFromDB(context.Background(), sess.ID())
	if err != nil {
		t.Fatalf("failed to load authenticated session: %v", err)
	}
	savedSession := session.NewSessionFromData(savedData)
	if !savedSession.IsAuthenticated() || savedSession.UserID() == nil || *savedSession.UserID() != foundUser.ID() {
		t.Fatal("expected saved session to be authenticated for registered user")
	}

	for _, header := range rec.Header().Values("Set-Cookie") {
		if strings.Contains(header, sess.ID()) {
			return
		}
	}
	t.Fatalf("expected Set-Cookie for session %q", sess.ID())
}

func TestLoginPage(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	sess := newTestSession(t, req)
	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()

	app.LoginPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `action="/login"`) || !strings.Contains(body, `method="POST"`) {
		t.Fatal("expected login form")
	}
	expectedCSRF := `<input type="hidden" name="csrf_token" value="` + sess.CSRFToken() + `">`
	if !strings.Contains(body, expectedCSRF) {
		t.Fatal("expected session CSRF token in form")
	}
}

func TestLoginPage_MissingSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	app.LoginPage(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestLoginPage_ShowsFlashError(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	sess := newTestSession(t, req)
	sess.SetFlash("error", "invalid credentials")
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()

	app.LoginPage(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "invalid credentials") {
		t.Fatal("expected login page to contain flash error")
	}
	if sess.HasValue("flash:error") {
		t.Fatal("expected flash error to be consumed")
	}
}

func TestLoginFlashFlow(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	loginPageHandler := publicChain(st, http.HandlerFunc(app.LoginPage))

	getReq := httptest.NewRequest(http.MethodGet, "/login", nil)
	getRec := httptest.NewRecorder()
	loginPageHandler.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET /login status %d, got %d", http.StatusOK, getRec.Code)
	}
	cookie := parseSessionCookie(t, getRec)
	sess := loadTestSession(t, st, cookie.Value)
	sess.SetFlash("error", "invalid credentials")
	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	if saved := loadTestSession(t, st, cookie.Value); !saved.HasValue("flash:error") {
		t.Fatal("expected login flash error to be saved")
	}

	finalReq := httptest.NewRequest(http.MethodGet, "/login", nil)
	finalReq.AddCookie(cookie)
	finalRec := httptest.NewRecorder()
	loginPageHandler.ServeHTTP(finalRec, finalReq)

	if finalRec.Code != http.StatusOK || !strings.Contains(finalRec.Body.String(), "invalid credentials") {
		t.Fatal("expected login page to show flash error")
	}
	if saved := loadTestSession(t, st, cookie.Value); saved.HasValue("flash:error") {
		t.Fatal("expected login flash error to be consumed")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(url.Values{
		"username": {"nonexistent"},
		"password": {"ValidPassword123!"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sess := newTestSession(t, req)
	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	addSessionCookieForTest(t, req, sess.ID())
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()

	app.Login(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatal("expected redirect to /login")
	}
	if message, ok := sess.GetFlash("error"); !ok || message != "invalid credentials" {
		t.Fatalf("expected invalid credentials flash, got %q", message)
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	passwordHash, err := zpass.NewPasswordHash("ValidPassword123!")
	if err != nil {
		t.Fatal(err)
	}
	newUser := user.CreateUser("1234567890123", "testuser", "Test User", "test@example.com", passwordHash)
	if err := st.CreateUser(context.Background(), newUser); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(url.Values{
		"username": {"testuser"},
		"password": {"WrongPassword123!"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sess := newTestSession(t, req)
	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	addSessionCookieForTest(t, req, sess.ID())
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()

	app.Login(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatal("expected redirect to /login")
	}
	if message, ok := sess.GetFlash("error"); !ok || message != "invalid credentials" {
		t.Fatalf("expected invalid credentials flash, got %q", message)
	}
}

func TestLogin_Success(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	pass := "ValidPassword123!"
	passwordHash, err := zpass.NewPasswordHash(pass)
	if err != nil {
		t.Fatal(err)
	}
	newUser := user.CreateUser("1234567890123", "testuser", "Test User", "test@example.com", passwordHash)
	if err := st.CreateUser(context.Background(), newUser); err != nil {
		t.Fatal(err)
	}

	initialReq := httptest.NewRequest(http.MethodGet, "/login", nil)
	sess := newTestSession(t, initialReq)
	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	oldSessionID := sess.ID()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(url.Values{
		"username": {"testuser"},
		"password": {pass},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addSessionCookieForTest(t, req, oldSessionID)
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()

	app.Login(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/user/home" {
		t.Fatal("expected successful login redirect")
	}
	if sess.ID() == oldSessionID || !sess.IsAuthenticated() || sess.UserID() == nil || *sess.UserID() != newUser.ID() {
		t.Fatal("expected new authenticated session")
	}
	if _, err := st.GetSessionDataFromDB(context.Background(), oldSessionID); !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("expected old session to be deleted, got %v", err)
	}
	savedSession := loadTestSession(t, st, sess.ID())
	if !savedSession.IsAuthenticated() || savedSession.UserID() == nil || *savedSession.UserID() != newUser.ID() {
		t.Fatal("expected saved authenticated session")
	}

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Value == sess.ID() {
			return
		}
	}
	t.Fatalf("expected Set-Cookie for session %q", sess.ID())
}

func TestLogout_Success(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	logoutUser := user.CreateUser("1234567890123", "logout-user", "Logout User", "logout@example.com", "password-hash")
	if err := st.CreateUser(context.Background(), logoutUser); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	data, err := session.GetSessionDataFromReq(req)
	if err != nil {
		t.Fatal(err)
	}
	userID := logoutUser.ID()
	data.UserID = &userID
	sess := session.NewSessionFromData(data)
	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	sessionID := sess.ID()
	req = session.SetSession(sess, req)
	addSessionCookieForTest(t, req, sessionID)
	rec := httptest.NewRecorder()

	app.Logout(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatal("expected logout redirect to /login")
	}
	if _, err := st.GetSessionDataFromDB(context.Background(), sessionID); !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("expected session to be deleted, got %v", err)
	}

	requestCookie := req.Cookies()[0]
	foundDeletedCookie := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name != requestCookie.Name {
			continue
		}
		foundDeletedCookie = true
		if cookie.Value != "" || cookie.MaxAge >= 0 {
			t.Fatalf("expected deleted session cookie, got value %q and MaxAge %d", cookie.Value, cookie.MaxAge)
		}
	}
	if !foundDeletedCookie {
		t.Fatal("expected session deletion cookie")
	}
}

func TestUserHome_Success(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	newUser := user.CreateUser("1234567890123", "testuser", "Test User", "test@example.com", "password-hash")
	if err := st.CreateUser(context.Background(), newUser); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)
	sess := newTestSession(t, req)
	if err := sess.Authenticate(newUser.ID(), req); err != nil {
		t.Fatal(err)
	}
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()
	app.UserHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Welcome, testuser!") || !strings.Contains(body, `action="/logout"`) || !strings.Contains(body, `method="POST"`) {
		t.Fatal("expected user home content and logout form")
	}
	expectedCSRF := `<input type="hidden" name="csrf_token" value="` + sess.CSRFToken() + `">`
	if !strings.Contains(body, expectedCSRF) {
		t.Fatal("expected session CSRF token in logout form")
	}
}

func TestUserHome_UserNotFound(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)
	sess := newTestSession(t, req)
	if err := sess.Authenticate(uuid.New(), req); err != nil {
		t.Fatal(err)
	}
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()
	app.UserHome(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestUserHome_MissingSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	rec := httptest.NewRecorder()
	app.UserHome(rec, httptest.NewRequest(http.MethodGet, "/user/home", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestUserHome_AnonymousSession(t *testing.T) {
	st := setupTestStore(t)
	app := newTestApplication(t, st)
	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)
	sess := newTestSession(t, req)
	req = session.SetSession(sess, req)
	rec := httptest.NewRecorder()
	app.UserHome(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
