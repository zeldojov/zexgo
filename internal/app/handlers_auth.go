package app

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/zeldojov/zexgo/internal/session"
	"github.com/zeldojov/zexgo/internal/user"
	"github.com/zeldojov/zexgo/internal/zpass"
)

func (a *application) LoginPage(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		a.InternalServerError(w, r)
		return
	}

	errorMessage, _ := sess.GetFlash("error")

	a.RenderStatus(w, http.StatusOK, "guest/login", struct {
		Title     string
		CSRFToken string
		Error     string
	}{
		Title:     "Login",
		CSRFToken: sess.CSRFToken(),
		Error:     errorMessage,
	})
}

func (a *application) Login(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		a.InternalServerError(w, r)
		return
	}

	username := r.PostFormValue("username")
	pass := r.PostFormValue("password")

	foundUser, err := a.store.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			sess.SetFlash("error", "invalid credentials")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		a.InternalServerError(w, r)
		return
	}

	if !foundUser.PasswordHash().Verify(pass) {
		sess.SetFlash("error", "invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := a.store.DeleteSession(r.Context(), sess.ID()); err != nil {
		a.InternalServerError(w, r)
		return
	}

	session.UnsetSessionCookie(w)

	if err := sess.Authenticate(foundUser.ID(), r); err != nil {
		a.InternalServerError(w, r)
		return
	}

	if err := a.store.SaveSession(r.Context(), sess); err != nil {
		a.InternalServerError(w, r)
		return
	}

	if err := session.SetSessionCookie(w, sess.ID()); err != nil {
		a.InternalServerError(w, r)
		return
	}

	http.Redirect(w, r, "/user/home", http.StatusSeeOther)
}

func (a *application) Logout(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		a.InternalServerError(w, r)
		return
	}

	if err := a.store.DeleteSession(r.Context(), sess.ID()); err != nil {
		a.InternalServerError(w, r)
		return
	}

	session.UnsetSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *application) RegisterPage(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		a.InternalServerError(w, r)
		return
	}

	errorMessage, _ := sess.GetFlash("error")

	a.RenderStatus(w, http.StatusOK, "guest/register", struct {
		Title     string
		CSRFToken string
		Error     string
	}{
		Title:     "Register",
		CSRFToken: sess.CSRFToken(),
		Error:     errorMessage,
	})
}

func (a *application) Register(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		a.InternalServerError(w, r)
		return
	}

	username := r.PostFormValue("username")
	pass := r.PostFormValue("password")
	jmbg := r.PostFormValue("jmbg")
	fullName := r.PostFormValue("full_name")
	email := r.PostFormValue("email")

	if err := user.ValidateUsername(username); err != nil {
		sess.SetFlash("error", "invalid username")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	if err := zpass.Validate(pass); err != nil {
		sess.SetFlash("error", "invalid password")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	passwordHash, err := zpass.NewPasswordHash(pass)
	if err != nil {
		a.InternalServerError(w, r)
		return
	}

	newUser := user.CreateUser(jmbg, username, fullName, email, passwordHash)

	if err := a.store.CreateUser(r.Context(), newUser); err != nil {
		if errors.Is(err, user.ErrUsernameTaken) {
			sess.SetFlash("error", "username already exists")
			http.Redirect(w, r, "/register", http.StatusSeeOther)
			return
		}

		a.InternalServerError(w, r)
		return
	}

	if err := a.store.DeleteSession(r.Context(), sess.ID()); err != nil {
		a.InternalServerError(w, r)
		return
	}

	session.UnsetSessionCookie(w)

	if err := sess.Authenticate(newUser.ID(), r); err != nil {
		a.InternalServerError(w, r)
		return
	}

	if err := a.store.SaveSession(r.Context(), sess); err != nil {
		a.InternalServerError(w, r)
		return
	}

	if err := session.SetSessionCookie(w, sess.ID()); err != nil {
		a.InternalServerError(w, r)
		return
	}

	http.Redirect(w, r, "/user/home", http.StatusSeeOther)
}

func (a *application) UserHome(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		a.InternalServerError(w, r)
		return
	}

	if sess.UserID() == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	foundUser, err := a.store.GetUserByID(r.Context(), *sess.UserID())
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		a.InternalServerError(w, r)
		return
	}

	_, _ = fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
	<title>User Home</title>
</head>
<body>
	<h1>Welcome, %s!</h1>

	<form action="/logout" method="POST">
		<input type="hidden" name="csrf_token" value="%s">
		<button type="submit">Logout</button>
	</form>
</body>
</html>
`, foundUser.Username(), sess.CSRFToken())
}
