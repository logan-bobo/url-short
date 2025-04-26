package configuration

import (
	"errors"
	"fmt"
	"os"
)

type ApplicationSettings struct {
	Server   *ServerSettings
	Database *DatabaseSettings
	Cache    *CacheSettings
}

func NewApplicationSettings() (*ApplicationSettings, error) {
	serverSettings, err := newServerSettings()
	if err != nil {
		return nil, err
	}
	databaseSettings, err := newDatabaseSettings()
	if err != nil {
		return nil, err
	}
	cacheSettings, err := newCacheSettings()
	if err != nil {
		return nil, err
	}

	return &ApplicationSettings{
		Server:   serverSettings,
		Database: databaseSettings,
		Cache:    cacheSettings,
	}, nil
}

type ServerSettings struct {
	Port      string
	JwtSecret string
}

func newServerSettings() (*ServerSettings, error) {
	serverPort, found := os.LookupEnv("APP_SERVER_PORT")
	if !found {
		return nil, errors.New(
			"could not build server settings: not found APP_SERVER_PORT",
		)
	}

	jwtSecret, found := os.LookupEnv("APP_JWT_SECRET")
	if !found {
		return nil,
			errors.New(
				"could not build server settings: not found APP_JWT_SECRET",
			)
	}

	serverSettings := ServerSettings{
		Port:      serverPort,
		JwtSecret: jwtSecret,
	}

	return &serverSettings, nil
}

type DatabaseSettings struct {
	host         string
	port         string
	databaseName string
	user         string
	password     string
	sslMode      string
}

func newDatabaseSettings() (*DatabaseSettings, error) {
	databaseHost, found := os.LookupEnv("APP_DB_HOST")
	if !found {
		return nil, errors.New(
			"could not build database settings: not found APP_DB_HOST",
		)
	}

	databasePort, found := os.LookupEnv("APP_DB_PORT")
	if !found {
		return nil, errors.New(
			"could not build database settings: not found APP_DB_PORT",
		)
	}

	databaseName, found := os.LookupEnv("APP_DB_NAME")
	if !found {
		return nil, errors.New(
			"could not build database settings: not found APP_DB_NAME",
		)
	}

	databaseUser, found := os.LookupEnv("APP_DB_USER")
	if !found {
		return nil, errors.New(
			"could not build database settings: not found APP_DB_USER",
		)
	}

	databasePassword, found := os.LookupEnv("APP_DB_PASSWORD")
	if !found {
		return nil, errors.New(
			"could not build database settings: not found APP_DB_PASSWORD",
		)
	}

	databaseSSLMode, found := os.LookupEnv("APP_DB_SSL_MODE")
	if !found {
		return nil, errors.New(
			"could not build server settings: not found APP_DB_SSL_MODE",
		)
	}

	databaseSettings := DatabaseSettings{
		host:         databaseHost,
		port:         databasePort,
		databaseName: databaseName,
		user:         databaseUser,
		password:     databasePassword,
		sslMode:      databaseSSLMode,
	}

	return &databaseSettings, nil
}

func (d *DatabaseSettings) GetPostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%v:%v@%v:%v/%v?sslmode=%v",
		d.user,
		d.password,
		d.host,
		d.port,
		d.databaseName,
		d.sslMode,
	)
}

func (d *DatabaseSettings) SetDatabaseName(dbName string) {
	d.databaseName = dbName
}

type CacheSettings struct {
	host       string
	port       string
	databaseId string
}

func newCacheSettings() (*CacheSettings, error) {
	cacheHost, found := os.LookupEnv("APP_CACHE_HOST")
	if !found {
		return nil, errors.New(
			"could not build cache settings: not found APP_DB_USER",
		)
	}

	cachePort, found := os.LookupEnv("APP_CACHE_PORT")
	if !found {
		return nil, errors.New(
			"could not build cache settings: not found APP_CACHE_PORT",
		)
	}

	cacheDatabaseID, found := os.LookupEnv("APP_CACHE_DB_ID")
	if !found {
		return nil, errors.New(
			"could not build cache settings: not found APP_CACHE_DB_ID",
		)
	}

	cacheSettings := CacheSettings{
		host:       cacheHost,
		port:       cachePort,
		databaseId: cacheDatabaseID,
	}

	return &cacheSettings, nil
}

func (c *CacheSettings) GetCacheURL() string {
	return fmt.Sprintf("redis://%v:%v/%v", c.host, c.port, c.databaseId)
}
