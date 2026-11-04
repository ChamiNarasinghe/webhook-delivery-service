
package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate executes the SQL migration file.
func Migrate(pool *pgxpool.Pool) error {
	sqlFile, err := os.ReadFile("migrations/001_create_tables.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	_, err = pool.Exec(context.Background(), string(sqlFile))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	fmt.Println("Database tables created successfully!")

	return nil
}
