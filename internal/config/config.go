package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppName  string
	HTTP     HTTPConfig
	Database DatabaseConfig
}

type HTTPConfig struct {
	Port uint16
}

type DatabaseConfig struct {
	Host     string
	Port     uint16
	User     string
	Password string
	Name     string
	SSLMode  string
}

func Load() (Config, error) {
	httpPort, err := portFromEnv("PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	databasePort, err := portFromEnv("DB_PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	databaseHost, err := requiredEnv("DB_HOST")

	if err != nil {
		return Config{}, err
	}

	databaseUser, err := requiredEnv("DB_USER")

	if err != nil {
		return Config{}, err
	}

	databasePassword, err := requiredEnv("DB_PASSWORD")

	if err != nil {
		return Config{}, err
	}

	databaseName, err := requiredEnv("DB_NAME")

	if err != nil {
		return Config{}, err
	}

	return Config{
			AppName: valueOrDefault("APP_NAME", "Question-To-Quiz"),
			HTTP: HTTPConfig{
				Port: httpPort,
			},

			Database: DatabaseConfig{
				Host:     databaseHost,
				Port:     databasePort,
				User:     databaseUser,
				Password: databasePassword,
				Name:     databaseName,
				SSLMode:  valueOrDefault("DB_SSLMODE", "disable"),
			},
		},
		nil

}

func requiredEnv(name string) (string, error) {
	value := os.Getenv(name)

	if value == "" {
		return "", fmt.Errorf("envorinment variable %s is requirred", name)
	}

	return value, nil
}

func valueOrDefault(name string, defaultName string) string {
	value := os.Getenv(name)

	if value == "" {
		return defaultName
	}
	return value
}

func portFromEnv(name string, defaultValue uint16) (uint16, error) {
	defaultPort := strconv.FormatUint(uint64(defaultValue), 10)
	rawPort := valueOrDefault(name, defaultPort)

	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil || port == 0 {
		return 0, fmt.Errorf(
			"environment variable %s must be a valid port, got %q",
			name,
			rawPort,
		)
	}

	return uint16(port), nil
}
