package postgres

import (
	"clic-metric/internal/storage"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", storagePath)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS url(
		id SERIAL PRIMARY KEY,
		alias TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL,
		clicks INT DEFAULT 0
	)
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = db.Exec(`
        	CREATE INDEX IF NOT EXISTS index_alias ON url(alias)
    	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.postgres.SaveURL"
	var id int64

	err := s.db.QueryRow(
		"INSERT INTO url(alias, url) VALUES ($1, $2) RETURNING id", alias, urlToSave,
	).Scan(&id)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUrlExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.postgres.GetUTL"

	stmt, err := s.db.Prepare("SELECT url FROM url WHERE alias = $1")

	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var resURL string
	err = stmt.QueryRow(alias).Scan(&resURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrUrlNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil

}

func (s *Storage) AddClick(alias string) error {
    _, err := s.db.Exec("UPDATE url SET clicks = clicks + 1 WHERE alias = $1", alias)
    return err
}

func (s *Storage) GetClicks(alias string) (int, error) {
    var clicks int
    err := s.db.QueryRow("SELECT clicks FROM url WHERE alias = $1", alias).Scan(&clicks)
    if errors.Is(err, sql.ErrNoRows) {
        return 0, storage.ErrUrlNotFound
    }
    return clicks, err
}