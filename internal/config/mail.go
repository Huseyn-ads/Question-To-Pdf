package config

import (
	"fmt"
	"net/mail"
	"os"
	"strings"
)

type MailConfig struct {
	Host     string
	Port     uint16
	Username string
	Password string
	From     string
	TLSMode  string
}

func loadMail(production bool) (MailConfig, error) {
	hostDefault, fromDefault, tlsDefault := "localhost", "no-reply@example.test", "none"
	portDefault := uint16(1025)
	if production {
		hostDefault, fromDefault, tlsDefault = "", "", "starttls"
		portDefault = 587
	}
	port, err := portFromEnv("SMTP_PORT", portDefault)
	if err != nil {
		return MailConfig{}, err
	}
	cfg := MailConfig{
		Host:     valueOrDefault("SMTP_HOST", hostDefault),
		Port:     port,
		Username: os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     valueOrDefault("SMTP_FROM", fromDefault),
		TLSMode:  valueOrDefault("SMTP_TLS", tlsDefault),
	}
	if cfg.Host == "" || strings.ContainsAny(cfg.Host, "\r\n /") {
		return MailConfig{}, fmt.Errorf("SMTP_HOST must be a hostname")
	}
	from, err := mail.ParseAddress(cfg.From)
	if err != nil || from.Name != "" || from.Address != cfg.From || strings.ContainsAny(cfg.From, "\r\n") {
		return MailConfig{}, fmt.Errorf("SMTP_FROM must be a bare email address")
	}
	if cfg.TLSMode != "none" && cfg.TLSMode != "starttls" && cfg.TLSMode != "tls" {
		return MailConfig{}, fmt.Errorf("SMTP_TLS must be none, starttls or tls")
	}
	if production && cfg.TLSMode == "none" {
		return MailConfig{}, fmt.Errorf("production SMTP requires TLS")
	}
	if (cfg.Username == "") != (cfg.Password == "") {
		return MailConfig{}, fmt.Errorf("SMTP_USER and SMTP_PASSWORD must both be set or both be empty")
	}
	if cfg.TLSMode == "none" && cfg.Username != "" {
		return MailConfig{}, fmt.Errorf("SMTP authentication requires TLS")
	}
	return cfg, nil
}
