# Question-To-Pdf
Service for students - pdf questions to quiz

HTTP, cookies, CSRF, environment settings and request limits:
[docs/api.md](docs/api.md).

Before running the backend, apply Goose migrations (including the session CSRF
migration). Copy `.env.example` to `.env` and replace the placeholder DB values.
For a public deployment set `APP_ENV=production`, an HTTPS `PUBLIC_ORIGIN`,
and run behind an HTTPS reverse proxy.

Checks: `go test ./...`, `go test -race ./...`, `go vet ./...`.

Email verification setup (Gmail SMTP or local Mailpit inbox), migration and
manual checks: [docs/email-verification.md](docs/email-verification.md).
