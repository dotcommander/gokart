// Example: PostgreSQL database operations with GoKart.
//
// This example demonstrates:
//   - Connection pooling with sensible defaults
//   - Query patterns (single row, multiple rows)
//   - Transaction helpers with auto-rollback
//   - Custom connection configuration
//
// Prerequisites:
//   - PostgreSQL running
//   - Set DATABASE_URL environment variable:
//     export DATABASE_URL="postgres://user:password@localhost:5432/mydb"
//
// To create a test table:
//   CREATE TABLE users (
//       id SERIAL PRIMARY KEY,
//       name TEXT NOT NULL,
//       email TEXT NOT NULL UNIQUE,
//       created_at TIMESTAMP DEFAULT NOW()
//   );
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dotcommander/gokart/postgres"
	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()

	// Example 1: Simple connection with default settings
	// Uses production-ready defaults:
	//   - MaxConns: 25
	//   - MinConns: 5
	//   - MaxConnLifetime: 1 hour
	//   - MaxConnIdleTime: 30 minutes
	//   - HealthCheckPeriod: 1 minute
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	pool, err := postgres.Open(ctx, url)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("Connected to PostgreSQL")

	// Example 2: Query single row
	var userName string
	var userEmail string
	err = pool.QueryRow(ctx, "SELECT name, email FROM users WHERE id = $1", 1).Scan(&userName, &userEmail)
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Println("No user found with id=1")
		} else {
			log.Printf("Query failed: %v", err)
		}
	} else {
		log.Printf("User: %s (%s)", userName, userEmail)
	}

	// Example 3: Query multiple rows
	rows, _ := pool.Query(ctx, "SELECT id, name, email FROM users LIMIT 10")
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

	// Example 4: Execute INSERT
	result, err := pool.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", "Bob", "bob@example.com")
	if err != nil {
		log.Printf("Insert failed: %v", err)
	} else {
		rowsAffected := result.RowsAffected()
		log.Printf("Inserted %d row(s)", rowsAffected)
	}

	// Example 5: Transaction with auto-rollback
	// If the function returns an error, the transaction is automatically rolled back
	err = postgres.Transaction(ctx, pool, func(tx pgx.Tx) error {
		// Insert user
		_, err := tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", "Charlie", "charlie@example.com")
		if err != nil {
			return fmt.Errorf("insert failed: %w", err)
		}

		// Update another record
		_, err = tx.Exec(ctx, "UPDATE users SET name = $1 WHERE id = $2", "Charlie Updated", 1)
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

	// Example 6: Custom connection configuration
	// Useful for high-traffic scenarios
	customPool, err := postgres.OpenWithConfig(ctx, postgres.Config{
		URL:               url,
		MaxConns:          50,  // Increase max connections
		MinConns:          10,  // Keep more warm connections
		MaxConnLifetime:   2 * time.Hour,
		MaxConnIdleTime:   15 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	})
	if err != nil {
		log.Printf("Custom config failed: %v", err)
	} else {
		defer customPool.Close()
		log.Println("Connected with custom config")
	}

	// Example 7: Access underlying pgxpool.Pool for advanced operations
	// GoKart doesn't hide the underlying types
	stats := pool.Stat()
	log.Printf("Pool stats: %d/%d connections (current/max)",
		stats.TotalConns(), stats.MaxConns())

	// Example 8: Prepared statements (for frequently executed queries)
	// Prepare once, execute many times
	// prepareErr := pool.Prepare(ctx, "getUser", "SELECT name, email FROM users WHERE id = $1")
	// if prepareErr == nil {
	//     row := pool.QueryRow(ctx, "getUser", 1)
	//     row.Scan(&userName, &userEmail)
	// }
}

// Example: Transaction with panic recovery
//
// WithTransaction also recovers from panics and rolls back:
//
//	err = postgres.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
//	    _, err := tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", "Dave", "dave@example.com")
//	    if err != nil {
//	        panic("database error!") // Transaction will be rolled back
//	    }
//	    return nil
//	})
//
// Example: Using WithTransaction for batch operations
//
//	err = postgres.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
//	    // Batch insert
//	    batch := &pgx.Batch{}
//	    for i := 0; i < 100; i++ {
//	        batch.Queue("INSERT INTO users (name, email) VALUES ($1, $2)",
//	            fmt.Sprintf("User%d", i),
//	            fmt.Sprintf("user%d@example.com", i))
//	    }
//
//	    br := tx.SendBatch(ctx, batch)
//	    defer br.Close()
//
//	    // Check results
//	    for i := 0; i < 100; i++ {
//	        _, err := br.Exec()
//	        if err != nil {
//	            return fmt.Errorf("batch insert failed at %d: %w", i, err)
//	        }
//	    }
//	    return nil
//	})
