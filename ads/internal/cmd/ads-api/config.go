package main

import (
	"os"
	"strconv"
	"time"
)

// Config holds everything the API needs to start.
type Config struct {
	// HTTPAddr is the address the API listens on, for example ":8080".
	HTTPAddr string
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string
	// IdentityURL is the base URL of the identity service.
	IdentityURL string
	// IdentityTimeout bounds every call to the identity service.
	IdentityTimeout time.Duration
	// MaxPublications is how many ads a user can publish through POST /ads within PublicationWindow.
	MaxPublications int
	// PublicationWindow is the rolling window MaxPublications applies to.
	PublicationWindow time.Duration
}

// Load reads the configuration from the environment.
func Load() Config {
	return Config{
		HTTPAddr:          text("HTTP_ADDR", ":8080"),
		DatabaseURL:       text("DATABASE_URL", "postgres://ads:ads@localhost:5432/ads?sslmode=disable"),
		IdentityURL:       text("IDENTITY_URL", "http://localhost:8081"),
		IdentityTimeout:   duration("IDENTITY_TIMEOUT", 2*time.Second),
		MaxPublications:   integer("MAX_PUBLICATIONS", 5),
		PublicationWindow: duration("PUBLICATION_WINDOW", 24*time.Hour),
	}
}

func integer(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return fallback
	}

	return value
}

func text(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}

func duration(name string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(name))
	if err != nil {
		return fallback
	}

	return value
}
