package questionimport

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"qtp/internal/httpx"
	"qtp/internal/user"
	"strings"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (handler *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	const (
		maxFileSize    = 10 << 20
		maxRequestSize = maxFileSize + (1 << 20)
	)

	currentUser, ok := user.UserFromContext(r.Context())
	if !ok {
		httpx.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	bankID := r.PathValue("bankID")

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	if fileHeader.Size == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "file is empty")
		return
	}

	if fileHeader.Size > maxFileSize {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "file is too large")
		return
	}

	originalName := filepath.Base(fileHeader.Filename)

	extension := strings.ToLower(filepath.Ext(originalName))

	if extension != ".docx" {
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "only DOCX files are supported")
		return
	}

	temporaryFile, err := os.CreateTemp("", "qtp-import-*.docx")
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "cannot create temporary file")
		return
	}

	temporaryPath := temporaryFile.Name()

	defer os.Remove(temporaryPath)

	_, copyErr := io.Copy(temporaryFile, file)

	closeErr := temporaryFile.Close()

	if copyErr != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "cannot save uploaded file")
		return
	}

	if closeErr != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "cannot close uploaded file")
		return
	}

	result, err := handler.service.Preview(r.Context(), currentUser.ID, bankID, temporaryPath)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)
}
