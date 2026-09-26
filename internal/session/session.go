package session

import (
	"errors"
	"maps"
	"net/http"
	"time"
	"uuid"

	"github.com/zeldojov/zexgo/internal/utils"
)

const (
	sessionDuration         = 30 * time.Minute
	sessionAbsoluteDuration = 7 * 24 * time.Hour
)

type (
	ContextKey struct{}

	Session struct {
		id        string
		csrfToken string
		userID    *uuid.UUID
		userIP    string
		userAgent string
		userData  map[string]string
		createdAt time.Time
		expiresAt time.Time
	}

	SessionData struct {
		ID        string
		CSRFToken string
		UserID    *uuid.UUID
		UserIP    string
		UserAgent string
		UserData  map[string]string
		CreatedAt time.Time
		ExpiresAt time.Time
	}
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

func NewSessionFromReq(r *http.Request) (*Session, error) {
	data, err := GetSessionDataFromReq(r)
	if err != nil {
		return nil, err
	}

	s := NewSessionFromData(data)
	s.Touch()

	return s, nil
}

func GetSessionDataFromReq(r *http.Request) (SessionData, error) {
	id, err := utils.NewRandomToken()
	if err != nil {
		return SessionData{}, err
	}

	csrfToken, err := utils.NewRandomToken()
	if err != nil {
		return SessionData{}, err
	}

	return SessionData{
		ID:        id,
		CSRFToken: csrfToken,
		UserID:    nil,
		UserIP:    utils.GetClientIP(r),
		UserAgent: utils.GetUserAgent(r),
		UserData:  map[string]string{},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionDuration),
	}, nil
}

func NewSessionFromData(data SessionData) *Session {
	var userID *uuid.UUID
	if data.UserID != nil {
		id := *data.UserID
		userID = &id
	}

	userData := make(map[string]string, len(data.UserData))
	maps.Copy(userData, data.UserData)

	return &Session{
		id:        data.ID,
		csrfToken: data.CSRFToken,
		userID:    userID,
		userIP:    data.UserIP,
		userAgent: data.UserAgent,
		userData:  userData,
		createdAt: data.CreatedAt,
		expiresAt: data.ExpiresAt,
	}
}

func SessionDuration() time.Duration {
	return sessionDuration
}

func (sess *Session) SetExpiresAt(when time.Time) {
	sess.expiresAt = when
}

func (sess *Session) SetUserIP(ip string) {
	sess.userIP = ip
}

func (sess *Session) SetUserAgent(agent string) {
	sess.userAgent = agent
}

func (sess *Session) ClearValues() {
	sess.userData = make(map[string]string)
}

func (sess *Session) Touch() {
	now := time.Now()

	if sess.createdAt.IsZero() {
		sess.createdAt = now
	}

	slidingExpiration := now.Add(sessionDuration)
	absoluteExpiration := sess.createdAt.Add(sessionAbsoluteDuration)

	if slidingExpiration.After(absoluteExpiration) {
		sess.expiresAt = absoluteExpiration
	} else {
		sess.expiresAt = slidingExpiration
	}
}

func (s *Session) MatchUserAgent(r *http.Request) bool {
	return s.userAgent == utils.GetUserAgent(r)
}

func (s *Session) MatchIP(r *http.Request) bool {
	return s.userIP == utils.GetClientIP(r)
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}

func (s *Session) ShouldRefresh() bool {
	if s.IsExpired() {
		return false
	}

	return time.Until(s.expiresAt) < sessionDuration/2
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) CSRFToken() string {
	return s.csrfToken
}

func (s *Session) UserID() *uuid.UUID {
	return s.userID
}

func (s *Session) UserIP() string {
	return s.userIP
}

func (s *Session) UserAgent() string {
	return s.userAgent
}

func (s *Session) UserData() map[string]string {
	data := make(map[string]string, len(s.userData))
	maps.Copy(data, s.userData)

	return data
}

func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}
