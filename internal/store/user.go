package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zeldojov/zexgo/internal/user"
	"github.com/zeldojov/zexgo/internal/zpass"
)

const (
	createUsersTableQuery = `
CREATE TABLE IF NOT EXISTS users (
	id CHAR(36) NOT NULL,
	jmbg CHAR(13) NOT NULL,
	username VARCHAR(16) NOT NULL,
	full_name VARCHAR(64) NOT NULL,
	email VARCHAR(254) NOT NULL,
	password_hash VARCHAR(255) NOT NULL,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	email_verified_at TIMESTAMP NULL,
	password_changed_at TIMESTAMP NOT NULL,
	last_login_at TIMESTAMP NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,

	PRIMARY KEY (id),
	CONSTRAINT uq_users_jmbg UNIQUE (jmbg),
	CONSTRAINT uq_users_username UNIQUE (username),
	CONSTRAINT uq_users_email UNIQUE (email)
);
`

	createUserQuery = `
INSERT INTO users (
	id,
	jmbg,
	username,
	full_name,
	email,
	password_hash,
	active,
	email_verified_at,
	password_changed_at,
	last_login_at,
	created_at,
	updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

	updateUserQuery = `
UPDATE users
SET
	full_name = ?,
	updated_at = $1
WHERE id = $2
`

	updateUsernameQuery = `
UPDATE users
SET
	username = $1,
	updated_at = $2
WHERE id = $3
`

	updateEmailQuery = `
UPDATE users
SET
	email = $1,
	updated_at = $2
WHERE id = $3
`

	updatePasswordQuery = `
UPDATE users
SET
	password_hash = $1,
	updated_at = $2
WHERE id = $3
`

	getUserByUsernameQuery = `
SELECT
	id,
	jmbg,
	username,
	full_name,
	email,
	password_hash,
	active,
	email_verified_at,
	password_changed_at,
	last_login_at,
	created_at,
	updated_at
FROM users
WHERE username = $1
`

	getUserByIDQuery = `
SELECT
	id,
	jmbg,
	username,
	full_name,
	email,
	password_hash,
	active,
	email_verified_at,
	password_changed_at,
	last_login_at,
	created_at,
	updated_at
FROM users
WHERE id = $1
`

	deleteUserQuery = `
DELETE FROM users
WHERE id = $1
`

	updateLastLoginQuery = `
UPDATE users
SET
	last_login_at = $1,
	updated_at = $2
WHERE id = $3
`
)

func (s *Store) CreateUser(ctx context.Context, u *user.User) error {
	_, err := s.db.ExecContext(
		ctx,
		createUserQuery,
		u.ID().String(),
		u.JMBG(),
		u.Username(),
		u.FullName(),
		u.Email(),
		u.PasswordHash(),
		u.Active(),
		u.EmailVerifiedAt(),
		u.PasswordChangedAt(),
		u.LastLoginAt(),
		u.CreatedAt(),
		u.UpdatedAt(),
	)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("create user: %w", err)
	}

	if pgErr.Code == "23505" {
		switch {
		case pgErr.ConstraintName == "uq_users_jmbg":
			return user.ErrJMBGTaken
		case pgErr.ConstraintName == "uq_users_username":
			return user.ErrUsernameTaken
		case pgErr.ConstraintName == "uq_users_email":
			return user.ErrEmailTaken
		}
	}

	return fmt.Errorf("create user: %w", err)
}

func (s *Store) UpdateUser(ctx context.Context, u *user.User) error {
	_, err := s.db.ExecContext(
		ctx,
		updateUserQuery,
		u.FullName(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		return fmt.Errorf("update user %q: %w", u.ID(), err)
	}

	return nil
}

func (s *Store) UpdateUsername(ctx context.Context, u *user.User) error {
	_, err := s.db.ExecContext(
		ctx,
		updateUsernameQuery,
		u.Username(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_users_username" {
			return user.ErrUsernameTaken
		}

		return fmt.Errorf("update username for user %q: %w", u.ID(), err)
	}

	return nil
}

func (s *Store) UpdateEmail(ctx context.Context, u *user.User) error {
	_, err := s.db.ExecContext(
		ctx,
		updateEmailQuery,
		u.Email(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_users_email" {
			return user.ErrEmailTaken
		}

		return fmt.Errorf("update email for user %q: %w", u.ID(), err)
	}

	return nil
}

func (s *Store) UpdatePassword(ctx context.Context, u *user.User) error {
	_, err := s.db.ExecContext(
		ctx,
		updatePasswordQuery,
		u.PasswordHash(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		return fmt.Errorf("update password for user %q: %w", u.ID(), err)
	}

	return nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var (
		userID            uuid.UUID
		jmbg              string
		username          string
		fullName          string
		email             string
		passwordHash      zpass.PasswordHash
		active            bool
		emailVerifiedAt   *time.Time
		passwordChangedAt time.Time
		lastLoginAt       *time.Time
		createdAt         time.Time
		updatedAt         time.Time
	)

	err := s.db.QueryRowContext(
		ctx,
		getUserByIDQuery,
		id.String(),
	).Scan(
		&userID,
		&jmbg,
		&username,
		&fullName,
		&email,
		&passwordHash,
		&active,
		&emailVerifiedAt,
		&passwordChangedAt,
		&lastLoginAt,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by id %q: %w", id, err)
	}

	return user.LoadUser(
		userID,
		jmbg,
		username,
		fullName,
		email,
		passwordHash,
		active,
		emailVerifiedAt,
		passwordChangedAt,
		lastLoginAt,
		createdAt,
		updatedAt,
	), nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*user.User, error) {
	var (
		userID            uuid.UUID
		userUsername      string
		jmbg              string
		fullName          string
		email             string
		passwordHash      zpass.PasswordHash
		active            bool
		emailVerifiedAt   *time.Time
		passwordChangedAt time.Time
		lastLoginAt       *time.Time
		createdAt         time.Time
		updatedAt         time.Time
	)

	err := s.db.QueryRowContext(
		ctx,
		getUserByUsernameQuery,
		username,
	).Scan(
		&userID,
		&jmbg,
		&userUsername,
		&fullName,
		&email,
		&passwordHash,
		&active,
		&emailVerifiedAt,
		&passwordChangedAt,
		&lastLoginAt,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by username %q: %w", username, err)
	}

	return user.LoadUser(
		userID,
		jmbg,
		userUsername,
		fullName,
		email,
		passwordHash,
		active,
		emailVerifiedAt,
		passwordChangedAt,
		lastLoginAt,
		createdAt,
		updatedAt,
	), nil
}

func (s *Store) DeleteUser(id uuid.UUID) error {
	result, err := s.db.Exec(
		deleteUserQuery,
		id.String(),
	)
	if err != nil {
		return fmt.Errorf("delete user %q: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user %q: %w", id, err)
	}

	if rows == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

func (s *Store) UpdateLastLogin(u *user.User) error {
	_, err := s.db.Exec(
		updateLastLoginQuery,
		u.LastLoginAt(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		return fmt.Errorf("update last login for user %q: %w", u.ID(), err)
	}

	return nil
}
