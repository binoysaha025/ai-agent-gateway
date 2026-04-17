package models

import (
	"database/sql"
	"time"
)

type APIKey struct {
	ID     int
	Key    string
	Name  string
	Plan string // free or pro
	CreatedAt time.Time
}

func CreateAPIKeyTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS api_keys (
		id SERIAL PRIMARY KEY,
		key VARCHAR(64) NOT NULL UNIQUE,
		name VARCHAR(100) NOT NULL,
		plan VARCHAR(10) NOT NULL DEFAULT 'free',
		created_at TIMESTAMP DEFAULT NOW()
	);`
	_, err := db.Exec(query)
	return err
}

func InsertAPIKey(db *sql.DB, key, name, plan string) error {
	query := `INSERT INTO api_keys (key, name, plan) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, key, name, plan)
	return err
}

func GetAPIKey(db *sql.DB, key string) (*APIKey, error) {
	row := db.QueryRow(`SELECT id, key, name, plan, created_at FROM api_keys WHERE key = $1`, key)
	k := &APIKey{}
	err := row.Scan(&k.ID, &k.Key, &k.Name, &k.Plan, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return k, err
}