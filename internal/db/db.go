package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const defaultDBPath = "/var/lib/kitin/kitin.db"

var DB *sql.DB

func Init() error {
	path, err := resolveDBPath()
	if err != nil {
		return err
	}

	var openErr error
	DB, openErr = sql.Open("sqlite3", path)
	if openErr != nil {
		return fmt.Errorf("could not open database: %v", openErr)
	}

	return createTables()
}

func resolveDBPath() (string, error) {
	if p := os.Getenv("KITIN_DB_PATH"); p != "" {
		return ensureDir(p)
	}

	if path, err := ensureDir(defaultDBPath); err == nil {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not create database directory for %s and no home directory fallback", defaultDBPath)
	}

	return ensureDir(filepath.Join(home, ".kitin", "kitin.db"))
}

func ensureDir(path string) (string, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("could not create database directory %s: %v", dir, err)
	}
	return path, nil
}

func createTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS schedules (
			id         TEXT PRIMARY KEY,
			token      TEXT NOT NULL,
			path       TEXT NOT NULL,
			every      TEXT NOT NULL,
			slack      TEXT,
			email      TEXT,
			discord    TEXT,
			last_run   DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS scans (
			id          TEXT PRIMARY KEY,
			token       TEXT NOT NULL,
			findings    TEXT,
			status      TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}
