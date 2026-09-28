package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(value)

	if err != nil {
		return err
	}
	return nil
}

func WriteError(w http.ResponseWriter, status int, message string) error {
	response := ErrorResponse{
		Error: message,
	}

	return WriteJSON(w, status, response)
}
