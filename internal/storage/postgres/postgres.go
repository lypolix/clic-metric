package postgres

import (
    "clic-metric/internal/domain"
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
    );
    CREATE TABLE IF NOT EXISTS events (
        id SERIAL PRIMARY KEY, 
        event_type TEXT NOT NULL, 
        payload TEXT NOT NULL, 
        status TEXT NOT NULL DEFAULT 'new' CHECK(status IN ('new', 'done')),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
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

func (s *Storage) SaveURL(urlToSave string, alias string) (id int64, err error) {
    const op = "storage.postgres.SaveURL"

    tx, err := s.db.Begin()
    if err != nil {
        return 0, fmt.Errorf("%s: %w", op, err)
    }

    defer func() {
        if err != nil {
            _ = tx.Rollback()
            return
        }

        commitErr := tx.Commit()
        if commitErr != nil {
            err = fmt.Errorf("%s: %w", op, commitErr)
        }
    }()

    err = tx.QueryRow(
        "INSERT INTO url(alias, url) VALUES ($1, $2) RETURNING id", alias, urlToSave,
    ).Scan(&id)

    if err != nil {
        if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
            return 0, fmt.Errorf("%s: %w", op, storage.ErrUrlExists)
        }
        return 0, fmt.Errorf("%s: %w", op, err)
    }

    eventPayload := fmt.Sprintf(
        `{"id": %d, "url": "%s", "alias": "%s"}`, 
        id, 
        urlToSave, 
        alias,
    )
    if err := s.saveEvent(tx, "URL CREATED", eventPayload); err != nil {
        return 0, fmt.Errorf("%s: %w", op, err)
    }
    return id, nil
}

func (s *Storage) saveEvent(tx *sql.Tx, eventType string, payload string) error {
    const op = "storage.postgres.saveEvent"

    stmt, err := tx.Prepare("INSERT INTO events(event_type, payload) VALUES ($1, $2)")
    if err != nil {
        return fmt.Errorf("%s: %w", op, err)
    }

    _, err = stmt.Exec(eventType, payload)
    if err != nil {
        return fmt.Errorf("%s: %w", op, err)
    }
    return nil
} 

type event struct {
    ID int `db:"id"`
    Type string `db:"event_type"`
    Payload string `db:"payload"`
}

func (s *Storage) GetNewEvent() (domain.Event, error) {
    const op = "storage.postgres.GetNewEvent"

    row := s.db.QueryRow("SELECT id, event_type, payload FROM events WHERE status = 'new' LIMIT 1")
    var evt event

    err := row.Scan(&evt.ID, &evt.Type, &evt.Payload) 
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return domain.Event{}, nil
        }
        return domain.Event{}, fmt.Errorf("%s: %w", op, err)
    }

    return domain.Event{
        ID: evt.ID,
        Type: evt.Type, 
        Payload: evt.Payload,
    }, nil
}

func (s *Storage) SetDone(id int) error {
    const op = "storage.postgres.MarkEventAsDone"

    stmt, err := s.db.Prepare("UPDATE events SET status = 'done' WHERE id = $1")
    if err != nil {
        return fmt.Errorf("%s: %w", op, err)
    }

    _, err = stmt.Exec(id)
    if err != nil {
        return fmt.Errorf("%s: %w", op, err)
    }

    return nil
}

func (s *Storage) GetURL(alias string) (string, error) {
    const op = "storage.postgres.GetURL"

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
