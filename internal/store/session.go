package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/zeldojov/zexgo/internal/session"
)

const (
	createSessionsTableQuery = `
CREATE TABLE IF NOT EXISTS sessions (
	id CHAR(64) NOT NULL PRIMARY KEY,
	csrf_token CHAR(64) NOT NULL,
	user_id CHAR(36) NULL,
	user_ip VARCHAR(45) NOT NULL,
	user_agent TEXT NOT NULL,
	user_data JSON NOT NULL,
	created_at TIMESTAMP NOT NULL,
	expires_at TIMESTAMP NOT NULL,

	FOREIGN KEY (user_id)
		REFERENCES users(id)
		ON DELETE CASCADE
);
`
	createSessionsUserIndexQuery = `
CREATE INDEX IF NOT EXISTS idx_sessions_user_id
ON sessions (user_id)
`

	saveSessionQuery = `
INSERT INTO sessions (
	id,
	csrf_token,
	user_id,
	user_ip,
	user_agent,
	user_data,
	created_at,
	expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE SET
	csrf_token = EXCLUDED.csrf_token,
	user_id = EXCLUDED.user_id,
	user_ip = EXCLUDED.user_ip,
	user_agent = EXCLUDED.user_agent,
	user_data = EXCLUDED.user_data,
	expires_at = EXCLUDED.expires_at
`

	getSessionByIDQuery = `
SELECT
	csrf_token,
	user_id,
	user_ip,
	user_agent,
	user_data,
	created_at,
	expires_at
FROM sessions
WHERE id = $1
`

	deleteSessionQuery = `
DELETE FROM sessions
WHERE id = $1
`
)

func (s *Store) SaveSession(ctx context.Context, sess *session.Session) error {
	data, err := json.Marshal(sess.UserData())
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		saveSessionQuery,
		sess.ID(),
		sess.CSRFToken(),
		sess.UserID(),
		sess.UserIP(),
		sess.UserAgent(),
		data,
		sess.CreatedAt(),
		sess.ExpiresAt(),
	)

	return err
}

func (s *Store) GetSessionDataFromDB(ctx context.Context, id string) (session.SessionData, error) {
	var (
		csrfToken string
		userID    sql.NullString
		dataJSON  string
		userIP    string
		userAgent string
		createdAt time.Time
		expiresAt time.Time
	)

	if err := s.db.QueryRowContext(ctx, getSessionByIDQuery, id).Scan(
		&csrfToken,
		&userID,
		&userIP,
		&userAgent,
		&dataJSON,
		&createdAt,
		&expiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return session.SessionData{}, session.ErrSessionNotFound
		}

		return session.SessionData{}, fmt.Errorf("scan session %q: %w", id, err)
	}

	if !time.Now().Before(expiresAt) {
		return session.SessionData{}, session.ErrSessionExpired
	}

	var userData map[string]string

	if err := json.Unmarshal([]byte(dataJSON), &userData); err != nil {
		return session.SessionData{}, fmt.Errorf("unmarshal session data %q: %w", id, err)
	}

	if userData == nil {
		userData = make(map[string]string)
	}

	var uid *uuid.UUID

	if userID.Valid {
		parsed, err := uuid.Parse(userID.String)
		if err != nil {
			return session.SessionData{}, fmt.Errorf("parse session user id %q: %w", userID.String, err)
		}

		uid = &parsed
	}

	return session.SessionData{
		ID:        id,
		CSRFToken: csrfToken,
		UserID:    uid,
		UserIP:    userIP,
		UserAgent: userAgent,
		UserData:  userData,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Store) NewSessionFromDB(ctx context.Context, id string) (*session.Session, error) {
	data, err := s.GetSessionDataFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	return session.NewSessionFromData(data), nil
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, deleteSessionQuery, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return session.ErrSessionNotFound
	}

	return nil
}
