package models

// marlow:ignore

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
