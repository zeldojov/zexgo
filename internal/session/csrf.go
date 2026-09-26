package session

import (
	"net/http"

	"github.com/zeldojov/zexgo/internal/utils"
)

const (
	csrfFieldName = "csrf_token"
)

func CSRFFieldName() string {
	return csrfFieldName
}

func GetCSRFToken(r *http.Request) (string, error) {
	session, ok := GetSession(r)
	if !ok || session == nil {
		return "", nil
	}

	if session.csrfToken == "" {
		token, err := utils.NewRandomToken()
		if err != nil {
			return "", err
		}

		session.csrfToken = token
	}

	return session.csrfToken, nil
}

func (s *Session) ValidateCSRFToken(r *http.Request) bool {
	if s.csrfToken == "" {
		return false
	}

	requestToken := r.PostFormValue(csrfFieldName)

	return requestToken != "" &&
		requestToken == s.csrfToken
}
