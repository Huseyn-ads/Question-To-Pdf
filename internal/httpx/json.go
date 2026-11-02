package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
)

const (
	MaxJSONBody   = 1 << 20
	MaxAuthBody   = 16 << 10
	MaxImportBody = 10 << 20
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) error {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return nil
	}
	// Encode before committing headers, so encoding failures cannot produce
	// a successful response with a partial JSON body.
	body, err := json.Marshal(value)
	if err != nil {
		slog.Error("encode HTTP response", "error", err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "{\"error\":\"server error\"}\n")
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err = w.Write(append(body, '\n'))
	if err != nil {
		slog.Error("write HTTP response", "error", err)
		return err
	}
	return nil
}

// ReadJSON accepts exactly one JSON object and never exposes decoder details.
// It writes a safe HTTP error and returns false if the request is invalid.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any, limit int64) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		_ = WriteError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	var raw json.RawMessage
	err = decoder.Decode(&raw)
	if err == nil {
		var extra json.RawMessage
		err = decoder.Decode(&extra)
		if errors.Is(err, io.EOF) {
			err = nil
		} else if err == nil {
			err = errors.New("multiple JSON values")
		}
	}
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		_ = WriteError(w, http.StatusRequestEntityTooLarge, "JSON body is too large")
		return false
	}
	raw = bytes.TrimSpace(raw)
	if err != nil || len(raw) == 0 || raw[0] != '{' {
		_ = WriteError(w, http.StatusBadRequest, "body must contain exactly one JSON object")
		return false
	}
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		_ = WriteError(w, http.StatusBadRequest, "invalid JSON body or unknown field")
		return false
	}
	return true
}

func WriteError(w http.ResponseWriter, status int, message string) error {
	response := ErrorResponse{
		Error: message,
	}

	return WriteJSON(w, status, response)
}
