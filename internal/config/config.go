package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppName  string
	HTTP     HTTPConfig
	Database DatabaseConfig
	Security SecurityConfig
}

type SecurityConfig struct {
	Production   bool
	PublicOrigin string
	CookieSecure bool
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

	security, err := loadSecurity(httpPort)
	if err != nil {
		return Config{}, err
	}

	databasePort, err := portFromEnv("DB_PORT", 5432)
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
			AppName:  valueOrDefault("APP_NAME", "Question-To-Quiz"),
			Security: security,
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

func loadSecurity(port uint16) (SecurityConfig, error) {
	env := valueOrDefault("APP_ENV", "development")
	if env != "development" && env != "test" && env != "production" {
		return SecurityConfig{}, fmt.Errorf("APP_ENV must be development, test or production")
	}
	production := env == "production"
	origin := strings.TrimRight(os.Getenv("PUBLIC_ORIGIN"), "/")
	if origin == "" {
		if production {
			return SecurityConfig{}, fmt.Errorf("PUBLIC_ORIGIN is required in production")
		}
		origin = "http://localhost:" + strconv.Itoa(int(port))
	}
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Path != "" ||
		u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") || strings.Contains(u.Host, "*") {
		return SecurityConfig{}, fmt.Errorf("PUBLIC_ORIGIN must be a single http(s) origin without a path")
	}
	if production && u.Scheme != "https" {
		return SecurityConfig{}, fmt.Errorf("production PUBLIC_ORIGIN must use HTTPS")
	}
	return SecurityConfig{Production: production, PublicOrigin: origin, CookieSecure: u.Scheme == "https"}, nil
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
