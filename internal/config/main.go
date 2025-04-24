package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Settings struct {
	application applicationSettings
	database    databaseSettings
	cache       cacheSettings
}

type applicationSettings struct {
	port int
	host string
}

type databaseSettings struct {
	port          int
	host          string
	username      string
	password      string
	database_name string
	require_ssql  bool
}

type cacheSettings struct {
	port     int
	host     string
	database int
	username string
	password string
}

func GetConfiguration() (Settings, error) {
	basePath, err := os.Getwd()
	if err != nil {
		return Settings{}, errors.New(fmt.Sprintf("failure to build configuration: %v", err.Error))
	}

	configPath := fmt.Sprintf("%v%v", basePath, "configuration")

	env, found := os.LookupEnv("APP_ENV")
	if !found {
		env = "local"
	}

	envFile := fmt.Sprintf("%v%V", env, ".yaml")

	viper.AddConfigPath(configPath)
	viper.SetConfigName(envFile)

	err = viper.ReadInConfig()
	if err != nil {
		return Settings{}, errors.New(fmt.Sprintf("failure to build configuration: %v", err.Error))
	}

	return settings, nil
}
