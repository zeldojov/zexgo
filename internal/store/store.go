package store

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrCreateUsersTable    = errors.New("failed to create users table")
	ErrCreateSessionsTable = errors.New("failed to create sessions table")
	errDatabaseOpen        = errors.New("failed to open postgres database connection")
	errDatabasePing        = errors.New("failed to ping postgres database connection")
)

type Store struct {
	db *sql.DB
}

type DBConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	SSLMode         string
}

func DefaultDBConfig() DBConfig {
	return DBConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    25,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		SSLMode:         "disable",
	}
}

func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}

	if _, err := s.db.Exec(createUsersTableQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateUsersTable, err)
	}

	if _, err := s.db.Exec(createSessionsTableQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateSessionsTable, err)
	}

	if _, err := s.db.Exec(createSessionsUserIndexQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateSessionsTable, err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func NewDSN(username, password, address, port, database string) string {
	return newDSN(username, password, address, port, database, DefaultDBConfig().SSLMode)
}

func newDSN(username, password, address, port, database, sslMode string) string {
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(address, port),
		Path:   "/" + database,
	}
	query := dsn.Query()
	query.Set("sslmode", sslMode)
	dsn.RawQuery = query.Encode()

	return dsn.String()
}

func Connect(username, password, address, port, database string) (*sql.DB, error) {
	return ConnectWithConfig(
		username,
		password,
		address,
		port,
		database,
		DefaultDBConfig(),
	)
}

func ConnectWithConfig(
	username,
	password,
	address,
	port,
	database string,
	config DBConfig,
) (*sql.DB, error) {
	dsn := newDSN(
		username,
		password,
		address,
		port,
		database,
		config.SSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDatabaseOpen, err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: %w", errDatabasePing, err)
	}

	return db, nil
}
