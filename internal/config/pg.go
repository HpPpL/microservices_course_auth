package config

import (
	"errors"
	"os"
)

const (
	dsnEnvName = "PG_DSN"
)

// PGConfig defines the interface for obtaining the PostgreSQL Data Source Name (DSN).
type PGConfig interface {
	DSN() string
}

type pgconfig struct {
	dsn string
}

// NewPGConfig creates a new PGConfig by reading the DSN from environment variables, returning an error if it's missing.
func NewPGConfig() (PGConfig, error) {
	dsn := os.Getenv(dsnEnvName)
	if len(dsn) == 0 {
		return nil, errors.New("pg dsn not found")
	}

	return &pgconfig{
		dsn: dsn,
	}, nil
}

func (cfg *pgconfig) DSN() string {
	return cfg.dsn
}
