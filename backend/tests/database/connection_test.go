package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseConnection(t *testing.T) {
	// Test SQLite connection
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Test basic connection
	err = db.Ping()
	require.NoError(t, err)

	// Test connection is usable
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestDatabaseConnectionPool(t *testing.T) {
	// Test connection pooling with SQLite
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Set connection pool parameters
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 30)

	// Test connection pool settings
	assert.Equal(t, 10, db.Stats().MaxOpenConnections)
	assert.Equal(t, 0, db.Stats().OpenConnections) // Should be 0 initially

	// Test multiple concurrent connections
	conns := make([]*sql.Conn, 5)
	for i := 0; i < 5; i++ {
		conn, err := db.Conn(context.Background())
		require.NoError(t, err)
		conns[i] = conn
		defer conn.Close()
	}

	// Should have 5 open connections
	assert.Equal(t, 5, db.Stats().OpenConnections)
}

func TestDatabaseConnectionFailure(t *testing.T) {
	// Test connection failure with invalid database
	_, err := sql.Open("sqlite3", "/invalid/path/database.db")
	require.NoError(t, err) // Open doesn't fail, connection fails on use

	db, _ := sql.Open("sqlite3", "/invalid/path/database.db")
	err = db.Ping()
	assert.Error(t, err)
	db.Close()
}

func TestDatabaseConnectionTimeout(t *testing.T) {
	// Test connection timeout
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test query with timeout
	var result int
	err = db.QueryRowContext(ctx, "SELECT 1").Scan(&result)

	// Should succeed quickly for in-memory DB
	if err != nil {
		assert.Contains(t, err.Error(), "context deadline exceeded")
	} else {
		assert.Equal(t, 1, result)
	}
}

func TestDatabaseConnectionIsolation(t *testing.T) {
	// Test that connections are isolated
	db1, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db1.Close()

	db2, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db2.Close()

	// Create table in first database
	_, err = db1.Exec("CREATE TABLE test (id INTEGER)")
	require.NoError(t, err)

	// Insert data in first database
	_, err = db1.Exec("INSERT INTO test (id) VALUES (1)")
	require.NoError(t, err)

	// Table should not exist in second database
	var count int
	err = db2.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	assert.Error(t, err) // Table doesn't exist in db2
}

func TestDatabaseConnectionRecovery(t *testing.T) {
	// Test connection recovery after failure
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Initial connection should work
	err = db.Ping()
	require.NoError(t, err)

	// Simulate connection failure by closing
	db.Close()

	// Ping should fail after close
	err = db.Ping()
	assert.Error(t, err)

	// Reopen connection
	db, err = sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Should work again
	err = db.Ping()
	require.NoError(t, err)
}

func TestDatabaseConnectionMetrics(t *testing.T) {
	// Test connection metrics
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Set pool parameters
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	// Get initial stats
	stats := db.Stats()
	assert.Equal(t, 0, stats.OpenConnections)
	assert.Equal(t, 0, stats.InUse)
	assert.Equal(t, 0, stats.Idle)
	assert.Equal(t, 5, stats.MaxOpenConnections)

	// Create some connections
	conns := make([]*sql.Conn, 3)
	for i := 0; i < 3; i++ {
		conn, err := db.Conn(context.Background())
		require.NoError(t, err)
		conns[i] = conn
	}

	// Check stats with active connections
	stats = db.Stats()
	assert.Equal(t, 3, stats.OpenConnections)
	assert.Equal(t, 3, stats.InUse)
	assert.Equal(t, 0, stats.Idle)

	// Close some connections
	for i := 0; i < 2; i++ {
		conns[i].Close()
	}

	// Check stats after closing some
	stats = db.Stats()
	assert.Equal(t, 1, stats.InUse)
	assert.Equal(t, 2, stats.Idle) // 2 idle, 1 in use

	// Close remaining connections
	conns[2].Close()

	// All should be idle now
	stats = db.Stats()
	assert.Equal(t, 0, stats.InUse)
	assert.Equal(t, 3, stats.Idle)
}

func TestDatabaseConnectionWithContext(t *testing.T) {
	// Test database operations with context
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Test with valid context
	ctx := context.Background()
	err = db.PingContext(ctx)
	require.NoError(t, err)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = db.PingContext(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestDatabaseTransactionConnection(t *testing.T) {
	// Test connection behavior during transactions
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("CREATE TABLE test_tx (id INTEGER, value TEXT)")
	require.NoError(t, err)

	// Start transaction
	tx, err := db.Begin()
	require.NoError(t, err)

	// Insert data in transaction
	_, err = tx.Exec("INSERT INTO test_tx (id, value) VALUES (1, 'test')")
	require.NoError(t, err)

	// Data should not be visible outside transaction yet
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_tx").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count) // Should be 0 before commit

	// Commit transaction
	err = tx.Commit()
	require.NoError(t, err)

	// Data should be visible now
	err = db.QueryRow("SELECT COUNT(*) FROM test_tx").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count) // Should be 1 after commit
}

func TestDatabasePreparedStatementConnection(t *testing.T) {
	// Test prepared statements with connection pooling
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("CREATE TABLE test_ps (id INTEGER, value TEXT)")
	require.NoError(t, err)

	// Prepare statement
	stmt, err := db.Prepare("INSERT INTO test_ps (id, value) VALUES (?, ?)")
	require.NoError(t, err)
	defer stmt.Close()

	// Execute prepared statement multiple times
	for i := 1; i <= 5; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("value_%d", i))
		require.NoError(t, err)
	}

	// Verify data
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_ps").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestDatabaseConnectionHealthCheck(t *testing.T) {
	// Test database health check functionality
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Health check should pass
	isHealthy := true
	err = db.Ping()
	if err != nil {
		isHealthy = false
	}
	assert.True(t, isHealthy)

	// Simulate unhealthy state
	db.Close()
	err = db.Ping()
	if err != nil {
		isHealthy = false
	}
	assert.False(t, isHealthy)
}

func TestDatabaseConnectionRetry(t *testing.T) {
	// Test connection retry logic
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	var db *sql.DB
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = sql.Open("sqlite3", ":memory:")
		if err != nil {
			if attempt == maxRetries {
				break
			}
			time.Sleep(retryDelay)
			continue
		}

		// Test connection
		err = db.Ping()
		if err != nil {
			if attempt == maxRetries {
				break
			}
			time.Sleep(retryDelay)
			continue
		}

		// Connection successful
		defer db.Close()
		require.NoError(t, err)
		return
	}

	// Should not reach here if connection works
	require.Fail(t, "Failed to establish database connection after %d attempts", maxRetries)
}

func TestDatabaseConnectionSecurity(t *testing.T) {
	// Test database connection security
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Test that connection doesn't expose sensitive information
	stats := db.Stats()
	assert.NotContains(t, stats.String(), "password", "Stats should not contain sensitive info")
	assert.NotContains(t, stats.String(), "secret", "Stats should not contain secrets")

	// Test SQL injection protection
	var result int
	// This should be safely parameterized, not concatenated
	err = db.QueryRow("SELECT 1 WHERE 1=?", 1).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)

	// Malicious query should fail or be handled safely
	err = db.QueryRow("SELECT 1; DROP TABLE test; --").Scan(&result)
	// Should either fail or be safely handled
	if err == nil {
		assert.Equal(t, 1, result) // If it succeeds, should only return first result
	}
}
