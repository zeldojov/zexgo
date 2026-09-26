package session

import (
	"net/http"
	"uuid"
)

func (s *Session) IsAuthenticated() bool {
	return s.userID != nil
}

func (s *Session) IsAnonymous() bool {
	return s.userID == nil
}

func (s *Session) Authenticate(userID uuid.UUID, r *http.Request) error {
	newSession, err := NewSessionFromReq(r)
	if err != nil {
		return err
	}

	newSession.userID = &userID
	*s = *newSession
	return nil
}
