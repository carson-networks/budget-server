package config

import (
	"fmt"
	"os"
)

type Config struct {
	PostgresAddress  string
	PostgresPort     string
	PostgresDB       string
	PostgresUsername string
	PostgresPassword string
	PlaidClientID    string
	PlaidSecret      string
	PlaidEnv         string // "sandbox", or "production"
}

func ProcessEnvironmentVariables() (*Config, error) {
	// In all cases the default behavior should be for the docker compose setup
	env := Config{
		PostgresAddress:  "localhost",
		PostgresPort:     "5433",
		PostgresDB:       "postgres",
		PostgresUsername: "postgres",
		PostgresPassword: "testpassword",
	}

	envPostgresAddress := os.Getenv("POSTGRES_ADDRESS")
	envPostgresPort := os.Getenv("POSTGRES_PORT")
	envPostgresDB := os.Getenv("POSTGRES_DB")
	envPostgresUsername := os.Getenv("POSTGRES_USERNAME")
	envPostgresPassword := os.Getenv("POSTGRES_PASSWORD")
	envPlaidClientID := os.Getenv("PLAID_CLIENT_ID")
	envPlaidSecret := os.Getenv("PLAID_SECRET")
	envPlaidEnv := os.Getenv("PLAID_ENVIRONMENT")

	if len(envPostgresAddress) != 0 {
		env.PostgresAddress = envPostgresAddress
	}

	if len(envPostgresPort) != 0 {
		env.PostgresPort = envPostgresPort
	}

	if len(envPostgresDB) != 0 {
		env.PostgresDB = envPostgresDB
	}

	if len(envPostgresUsername) != 0 {
		env.PostgresUsername = envPostgresUsername
	}

	if len(envPostgresPassword) != 0 {
		env.PostgresPassword = envPostgresPassword
	}

	if len(envPlaidClientID) != 0 {
		env.PlaidClientID = envPlaidClientID
	}

	if len(envPlaidSecret) != 0 {
		env.PlaidSecret = envPlaidSecret
	}

	if len(envPlaidEnv) != 0 {
		env.PlaidEnv = envPlaidEnv
	} else {
		env.PlaidEnv = "sandbox"
	}

	if env.PlaidClientID == "" || env.PlaidSecret == "" {
		return nil, fmt.Errorf("PLAID_CLIENT_ID and PLAID_SECRET must both be set")
	}

	return &env, nil
}
