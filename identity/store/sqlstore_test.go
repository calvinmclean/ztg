package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/calvinmclean/ztg/gen/go/sqlc"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// setupTestStore creates a PostgreSQL database for testing using pgtestdb.
//
// pgtestdb uses template databases to give each test a fully prepared and migrated
// database. Migrations from ../../migrations are run once and each test gets its own
// isolated database cloned from the template.
//
// Requirements:
//   - PostgreSQL server running on localhost:5432
//   - User "ztg" with password "password"
//   - SUPERUSER, CREATEDB, and CREATEROLE capabilities
func setupTestStore(t *testing.T) *sql.DB {
	t.Helper()

	// Configure pgtestdb to connect to test postgres server
	conf := pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		Port:       "5432",
		User:       "ztg",
		Password:   "password",
		Options:    "sslmode=disable",
	}

	// Create migrator using golang-migrate migrations
	migrator := golangmigrator.New("../../migrations")

	db := pgtestdb.New(t, conf, migrator)

	return db
}

func TestInsertIdentity(t *testing.T) {
	db := setupTestStore(t)
	defer db.Close()

	ctx := context.Background()

	// Test inserting identity
	queries := sqlc.New(db)

	params := sqlc.InsertIdentityParams{
		PublicKey:     []byte("test-public-key"),
		ServerAddress: "test.example.com:8080",
		ServerName:    "Test Server",
		OwnerName:     "Test Owner",
		Capabilities:  `["game1","game2"]`,
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
	if row.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if row.LastSeen.IsZero() {
		t.Error("LastSeen should not be zero")
	}
}
