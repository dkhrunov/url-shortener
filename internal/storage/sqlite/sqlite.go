package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/dkhrunov/url-shortener/internal/lib/zero"
	"github.com/dkhrunov/url-shortener/internal/storage"
	"github.com/mattn/go-sqlite3"
)

type Sqlite struct {
	db *sql.DB
}

func New(storagePath string) (*Sqlite, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`--sql
		CREATE TABLE IF NOT EXISTS url(
			id INTEGER PRIMARY KEY,
			alias TEXT NOT NULL UNIQUE,
			url TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Sqlite{db: db}, nil
}

func (s *Sqlite) SaveURL(urlToSave, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	stmt, err := s.db.Prepare(`--sql
		INSERT INTO url(url, alias) VALUES(?, ?)
	`)
	if err != nil {
		return zero.Zero[int64](), fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.Exec(urlToSave, alias)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return zero.Zero[int64](), fmt.Errorf("%s: %w", op, storage.ErrURLExist)
		}

		return zero.Zero[int64](), fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Sqlite) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	stmt, err := s.db.Prepare(`--sql
		SELECT url FROM url WHERE alias = ?
	`)
	if err != nil {
		return zero.Zero[string](), fmt.Errorf("%s: %w", op, err)
	}

	var resURL string
	err = stmt.QueryRow(alias).Scan(&resURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return zero.Zero[string](), fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
		}

		return zero.Zero[string](), fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil
}

func (s *Sqlite) DeleteURL(alias string) error {
	const op = "storage.sqlite.DeleteURL"

	stmt, err := s.db.Prepare(`--sql
		DELETE FROM url WHERE alias = ?
	`)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(alias)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
