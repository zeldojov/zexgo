package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/zeldojov/zexgo/internal/session"
)

func TestLogout_Success(t *testing.T) {
	st := setupTestStore(t)
	handler := NewHandler(st, nil, nil)

	userID := uuid.New()

	req := httptest.NewRequest(
		http.MethodPost,
		"/logout",
		nil,
	)

	data, err := session.GetSessionDataFromReq(req)
	if err != nil {
		t.Fatal(err)
	}

	data.UserID = &userID
	sess := session.NewSessionFromData(data)

	if err := st.SaveSession(context.Background(), sess); err != nil {
		t.Fatal(err)
	}

	sessionID := sess.ID()

	// Handler očekuje session u contextu.
	ctx := context.WithValue(
		req.Context(),
		session.ContextKey{},
		sess,
	)

	req = req.WithContext(ctx)

	// Handler očekuje session cookie.
	cookieRecorder := httptest.NewRecorder()
	if err := session.SetSessionCookie(cookieRecorder, sessionID); err != nil {
		t.Fatal(err)
	}
	req.AddCookie(cookieRecorder.Result().Cookies()[0])

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	// Logout mora da redirectuje na login.
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

	// Session mora biti obrisana iz DB-a.
	_, err = st.GetSessionDataFromDB(context.Background(), sessionID)

	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf(
			"expected session to be deleted, got error %v",
			err,
		)
	}

	// Cookie mora biti poništen.
	cookies := rec.Result().Cookies()

	var foundDeletedCookie bool
	cookieName := cookieRecorder.Result().Cookies()[0].Name

	for _, cookie := range cookies {
		if cookie.Name == cookieName {
			foundDeletedCookie = true

			if cookie.Value != "" {
				t.Fatalf(
					"expected deleted session cookie to have empty value, got %q",
					cookie.Value,
				)
			}

			if cookie.MaxAge >= 0 {
				t.Fatalf(
					"expected deleted session cookie to have negative MaxAge, got %d",
					cookie.MaxAge,
				)
			}

			break
		}
	}

	if !foundDeletedCookie {
		t.Fatal("expected session deletion cookie")
	}
}
