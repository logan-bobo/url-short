package main

import (
	"log"

	_ "github.com/lib/pq"

	"url-short/internal/application"
	"url-short/internal/configuration"
)

func main() {
	appSettings, err := configuration.NewApplicationSettings()
	if err != nil {
		log.Fatal("could not build application settings", err)
	}

	application, err := application.NewApplication(appSettings)
	if err != nil {
		log.Fatal("could not build application", err)
	}

	log.Printf("Serving port: %v \n", appSettings.Server.Port)
	log.Fatal(application.Server.ListenAndServe())
}
