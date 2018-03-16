package models

// marlow:ignore

// DatabaseConfig is a convenience type for storing the configuration necessary for the example app db connections.
type DatabaseConfig struct {
	Postgres struct {
		Username string
		Password string
		Database string
		Hostname string
		Port     string
	}
	SQLite struct {
		Filename string
	}
}
