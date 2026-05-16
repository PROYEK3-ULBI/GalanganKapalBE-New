// CLI utility to verify both DATABASE_URL and DATABASE_MIGRATION_URL are reachable.
// Useful for debugging connection issues without starting the full server.
//
//	go run ./cmd/dbcheck
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	urls := map[string]string{
		"DATABASE_URL (runtime, port 6543)":         os.Getenv("DATABASE_URL"),
		"DATABASE_MIGRATION_URL (migration, 5432)": os.Getenv("DATABASE_MIGRATION_URL"),
	}

	for label, u := range urls {
		fmt.Printf("\n[%s]\n", label)
		if u == "" {
			fmt.Println("  ✗ not set")
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		conn, err := pgx.Connect(ctx, u)
		if err != nil {
			cancel()
			fmt.Printf("  ✗ connect error: %v\n", err)
			continue
		}
		var version string
		if err := conn.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
			fmt.Printf("  ✗ query error: %v\n", err)
		} else {
			fmt.Printf("  ✓ connected\n  ✓ %s\n", version[:60])
		}
		conn.Close(ctx)
		cancel()
	}
}
