package user

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"html/template"
	"net/http"

	"qtp/internal/httpx"
)

//go:embed verification_page.html
var verificationHTML string

var verificationPage = template.Must(template.New("verify-email").Parse(verificationHTML))

func (handler *UserHandler) VerificationPage(w http.ResponseWriter, r *http.Request) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "server error")
		return
	}
	nonce := base64.RawStdEncoding.EncodeToString(random[:])
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'nonce-"+nonce+"'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
	_ = verificationPage.Execute(w, struct{ Nonce string }{Nonce: nonce})
}
