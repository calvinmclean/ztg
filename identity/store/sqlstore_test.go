package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/calvinmclean/ztg/gen/go/sqlc"
)

func TestInsertIdentity(t *testing.T) {
	// Create in-memory SQLite database using Turso driver
	db, err := sql.Open("turso", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Run migration
	err = runMigrations(ctx, db)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test inserting identity
	queries := sqlc.New(db)

	now := time.Now()
	params := sqlc.InsertIdentityParams{
		PublicKey:     []byte("test-public-key"),
		ServerAddress: "test.example.com:8080",
		ServerName:    "Test Server",
		OwnerName:     "Test Owner",
		Capabilities:  `["game1","game2"]`,
		CreatedAt:     now.Unix(),
		LastSeen:      now.Unix(),
		IsTrusted:     false,
	}

	result, err := queries.InsertIdentity(ctx, params)
	if err != nil {
		t.Fatalf("Failed to insert identity: %v", err)
	}

	t.Logf("Insert result: %v", result)

	// Test querying the identity
	row, err := queries.GetIdentityByKey(ctx, []byte("test-public-key"))
	if err != nil {
		t.Fatalf("Failed to get identity: %v", err)
	}

	t.Logf("Retrieved identity: CreatedAt=%v (type=%T), LastSeen=%v (type=%T)",
		row.CreatedAt, row.CreatedAt, row.LastSeen, row.LastSeen)

	// Verify the timestamp fields are properly set
	if row.CreatedAt == 0 {
		t.Error("CreatedAt should not be zero")
	}
	if row.LastSeen == 0 {
		t.Error("LastSeen should not be zero")
	}
}
