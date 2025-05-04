package helpers

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"os"
	"time"

	"github.com/pressly/goose/v3"

	"url-short/internal/configuration"
)

func generateRandomAlphaString(length int) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, length)
	for i := range result {
		result[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(result)
}

func migrateDB(db *sql.DB) error {
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		os.DirFS("./sql/schema/"),
	)

	if err != nil {
		return errors.New("can not create goose provider")
	}

	_, err = provider.Up(context.Background())

	if err != nil {
		return errors.New("could not run migrations")
	}

	return nil
}

func WithDB() (*sql.DB, error) {
	applicationSettings, err := configuration.NewApplicationSettings()
	if err != nil {
		return nil, err
	}

	dbMain, err := sql.Open("postgres", applicationSettings.Database.GetPostgresDSN())
	if err != nil {
		return nil, err
	}

	defer dbMain.Close()

	databaseName := generateRandomAlphaString(10)

	_, err = dbMain.Exec("create database " + databaseName)
	if err != nil {
		return nil, err
	}

	applicationSettings.Database.SetDatabaseName(databaseName)

	newDSN := applicationSettings.Database.GetPostgresDSN()

	testdb, err := sql.Open("postgres", newDSN)
	if err != nil {
		return nil, err
	}

	if err = migrateDB(testdb); err != nil {
		return nil, err
	}

	return testdb, nil
}
