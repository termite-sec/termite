package db

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init() error {
	var err error
	DB, err = sql.Open("sqlite3", "/var/lib/termite/termite.db")
	if err != nil {
		return fmt.Errorf("could not open database: %v", err)
	}

	return createTables()
}

func createTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS schedules (
			id          TEXT PRIMARY KEY,
			user        TEXT NOT NULL,
			path        TEXT NOT NULL,
			every       TEXT NOT NULL,
			cron        TEXT NOT NULL,
			slack       TEXT,
			email       TEXT,
			discord     TEXT,
			last_run    DATETIME,
			next_run    DATETIME,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS scans (
			id          TEXT PRIMARY KEY,
			user        TEXT NOT NULL,
			path        TEXT NOT NULL,
			status      TEXT NOT NULL,
			findings    TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			finished_at DATETIME
		);
	`)
	return err
}
