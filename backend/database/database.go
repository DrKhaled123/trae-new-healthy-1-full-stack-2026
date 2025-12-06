package database

import (
	"database/sql"
)

// Database wraps sql.DB to provide a consistent interface
type Database struct {
	DB *sql.DB
}

// NewDatabase creates a new Database wrapper
func NewDatabase(db *sql.DB) *Database {
	return &Database{DB: db}
}

// QueryRow executes a query that returns at most one row
func (d *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.DB.QueryRow(query, args...)
}

// Query executes a query that returns rows
func (d *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.DB.Query(query, args...)
}

// Exec executes a query without returning rows
func (d *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.DB.Exec(query, args...)
}

// Begin starts a transaction
func (d *Database) Begin() (*sql.Tx, error) {
	return d.DB.Begin()
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.DB.Close()
}

// Ping verifies the connection to the database
func (d *Database) Ping() error {
	return d.DB.Ping()
}
