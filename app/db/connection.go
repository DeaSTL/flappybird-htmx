package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

func NewConnection() (*Queries, error) {
	// Check if running in Docker (data directory exists) or locally
	db_filename := "/data/main.db"
	if _, err := os.Stat("/data"); os.IsNotExist(err) {
		// Running locally, use relative path
		db_filename = "../data/main.db"
	}

	db, err := sql.Open("sqlite3", db_filename)

	if err != nil {
		return nil, err
	}

	schema_file, err := os.ReadFile("./db/schema.sql")

	if err != nil {
		return nil, err
	}

	_, err = db.Exec(string(schema_file))

	if err != nil {
		return nil, err
	}

	queries := New(db)

	return queries, nil
}
