package config

import (
	"github.com/joho/godotenv"
)

// Load loads environment variables from the file at the specified path using
func Load(path string) error {
	err := godotenv.Load(path)
	if err != nil {
		return err
	}

	return nil
}
