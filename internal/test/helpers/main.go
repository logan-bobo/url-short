package helpers

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func ResetDB(db *sql.DB, schemaPath string) error {
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		os.DirFS(schemaPath),
	)

	if err != nil {
		return err
	}

	_, err = provider.DownTo(context.Background(), 0)

	if err != nil {
		return err
	}

	_, err = provider.Up(context.Background())

	if err != nil {
		return err
	}

	return nil
}

func NewTestDB() (uuid.UUID, error) {
	databaseId := uuid.Must(uuid.NewRandom())

	dbURL := os.Getenv("PG_CONN")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		return databaseId, err
	}

	query := fmt.Sprintf("CREATE DATABASE %s WITH OWNER url_short",
		pq.QuoteIdentifier(databaseId.String()))

	_, err = db.ExecContext(context.TODO(), query)

	if err != nil {
		return databaseId, err
	}

	defer db.Close()

	return databaseId, nil

}
