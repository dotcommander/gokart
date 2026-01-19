// Example: SQLite database operations with GoKart.
//
// This example demonstrates:
//   - Zero-CGO SQLite (pure Go implementation)
//   - Connection pooling with sensible defaults
//   - Transaction helpers with auto-rollback
//   - In-memory databases for testing
//
// Prerequisites:
//   - None (uses file-based database)
//   - Modernc.org/sqlite provides zero-CGO SQLite
//
// To create a test table:
//   CREATE TABLE users (
//       id INTEGER PRIMARY KEY AUTOINCREMENT,
//       name TEXT NOT NULL,
//       email TEXT NOT NULL UNIQUE,
//       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
//   );
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/dotcommander/gokart/sqlite"
)

func main() {
	ctx := context.Background()

	// Example 1: Simple connection with default settings
	// Uses production-ready defaults:
	//   - WAL mode: enabled (better concurrency)
	//   - Foreign keys: enabled
	//   - Busy timeout: 5 seconds
	//   - Max open connections: 25
	//   - Max idle connections: 5
	//   - Conn max lifetime: 1 hour
	db, err := sqlite.Open("app.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to SQLite (app.db)")

	// Example 2: Create table
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	log.Println("Table created (if not exists)")

	// Example 3: Query single row
	var userName string
	var userEmail string
	err = db.QueryRowContext(ctx, "SELECT name, email FROM users WHERE id = ?", 1).Scan(&userName, &userEmail)
	if err == sql.ErrNoRows {
		log.Println("No user found with id=1")
	} else if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		log.Printf("User: %s (%s)", userName, userEmail)
	}

	// Example 4: Query multiple rows
	rows, _ := db.QueryContext(ctx, "SELECT id, name, email FROM users LIMIT 10")
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, email string
		if err := rows.Scan(&id, &name, &email); err != nil {
			log.Printf("Row scan failed: %v", err)
			continue
		}
		log.Printf("User %d: %s <%s>", id, name, email)
	}

	// Example 5: Execute INSERT
	result, err := db.ExecContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@example.com")
	if err != nil {
		log.Printf("Insert failed: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		log.Printf("Inserted %d row(s)", rowsAffected)

		// Get last insert ID
		lastID, _ := result.LastInsertId()
		log.Printf("Last insert ID: %d", lastID)
	}

	// Example 6: Transaction with auto-rollback
	// If the function returns an error, the transaction is automatically rolled back
	err = sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
		// Insert user
		_, err := tx.ExecContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@example.com")
		if err != nil {
			return fmt.Errorf("insert failed: %w", err)
		}

		// Update another record
		_, err = tx.ExecContext(ctx, "UPDATE users SET name = ? WHERE id = ?", "Bob Updated", 1)
		if err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

		// If we get here, both operations succeeded, so commit
		return nil
	})

	if err != nil {
		log.Printf("Transaction failed (rolled back): %v", err)
	} else {
		log.Println("Transaction committed successfully")
	}

	// Example 7: Custom connection configuration
	customDB, err := sqlite.OpenWithConfig(ctx, sqlite.Config{
		Path:         "custom.db",
		WALMode:      true,
		ForeignKeys:  true,
		BusyTimeout:  10 * time.Second,
		MaxOpenConns: 50,
		MaxIdleConns: 10,
	})
	if err != nil {
		log.Printf("Custom config failed: %v", err)
	} else {
		defer customDB.Close()
		log.Println("Connected with custom config (custom.db)")
	}

	// Example 8: In-memory database (useful for tests)
	memDB, err := sqlite.InMemory()
	if err != nil {
		log.Printf("In-memory DB failed: %v", err)
	} else {
		defer memDB.Close()

		// Create table and insert data
		memDB.ExecContext(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
		memDB.ExecContext(ctx, "INSERT INTO test (value) VALUES (?)", "in-memory data")

		var value string
		memDB.QueryRowContext(ctx, "SELECT value FROM test WHERE id = 1").Scan(&value)
		log.Printf("In-memory DB value: %s", value)
	}

	// Example 9: Access underlying sql.DB for advanced operations
	// GoKart doesn't hide the underlying types
	stats := db.Stats()
	log.Printf("Pool stats: %d/%d open connections (in use/idle)",
		stats.InUse, stats.Idle)
}

// Example: Transaction with panic recovery
//
// WithTransaction also recovers from panics and rolls back:
//
//	err = sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
//	    _, err := tx.ExecContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "Charlie", "charlie@example.com")
//	    if err != nil {
//	        panic("database error!") // Transaction will be rolled back
//	    }
//	    return nil
//	})
//
// Example: Batch operations with transaction
//
//	err = sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
//	    stmt, err := tx.PrepareContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)")
//	    if err != nil {
//	        return err
//	    }
//	    defer stmt.Close()
//
//	    for i := 0; i < 100; i++ {
//	        _, err := stmt.ExecContext(ctx, fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@example.com", i))
//	        if err != nil {
//	            return fmt.Errorf("batch insert failed at %d: %w", i, err)
//	        }
//	    }
//	    return nil
//	})
//
// Example: Check SQLite capabilities
//
//	// Check if WAL mode is enabled
//	var walMode string
//	db.QueryRow("PRAGMA journal_mode").Scan(&walMode)
//	log.Printf("WAL mode: %s", walMode)
//
//	// Check foreign keys setting
//	var fkEnabled int
//	db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
//	log.Printf("Foreign keys: %d", fkEnabled)
