package questionimport

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"qtp/internal/httpx"
	"qtp/internal/importer"
	"qtp/internal/questionbank"
	"qtp/internal/user"
	"strings"
)

type Handler struct {
	service *Service
	slots   chan struct{}
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
		slots:   make(chan struct{}, 4),
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
	if err := handler.service.CheckBank(r.Context(), currentUser.ID, bankID); err != nil {
		handler.writeError(w, err)
		return
	}
	select {
	case handler.slots <- struct{}{}:
		defer func() { <-handler.slots }()
	default:
		httpx.WriteError(w, http.StatusTooManyRequests, "import capacity reached; try again later")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, "upload body is too large")
		} else {
			httpx.WriteError(w, http.StatusBadRequest, "invalid multipart form")
		}
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
		slog.Error("create import temporary file", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "cannot create temporary file")
		return
	}

	temporaryPath := temporaryFile.Name()

	defer os.Remove(temporaryPath)

	copied, copyErr := io.Copy(temporaryFile, io.LimitReader(file, maxFileSize+1))

	closeErr := temporaryFile.Close()

	if copyErr != nil {
		slog.Error("copy import temporary file", "error", copyErr)
		httpx.WriteError(w, http.StatusInternalServerError, "cannot save uploaded file")
		return
	}

	if closeErr != nil {
		slog.Error("close import temporary file", "error", closeErr)
		httpx.WriteError(w, http.StatusInternalServerError, "cannot close uploaded file")
		return
	}
	if copied > maxFileSize {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "file is too large")
		return
	}
	if r.Context().Err() != nil {
		return
	}

	result, err := handler.service.Preview(r.Context(), currentUser.ID, bankID, temporaryPath)
	if err != nil {
		handler.writeError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)
}

func (handler *Handler) Import(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest

	currentUser, ok := user.UserFromContext(r.Context())

	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	bankID := r.PathValue("bankID")
	if err := handler.service.CheckBank(r.Context(), currentUser.ID, bankID); err != nil {
		handler.writeError(w, err)
		return
	}
	select {
	case handler.slots <- struct{}{}:
		defer func() { <-handler.slots }()
	default:
		httpx.WriteError(w, http.StatusTooManyRequests, "import capacity reached; try again later")
		return
	}
	if !httpx.ReadJSON(w, r, &req, httpx.MaxImportBody) {
		return
	}

	importResponse, err := handler.service.Import(r.Context(), currentUser.ID, bankID, req)

	if err != nil {
		switch {
		case errors.Is(err, questionbank.ErrInvalidQuestionBankID):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				questionbank.ErrInvalidQuestionBankID.Error(),
			)

		case errors.Is(err, questionbank.ErrQuestionBankNotFound):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				questionbank.ErrQuestionBankNotFound.Error(),
			)

		default:
			slog.Error("import questions", "error", err)
			httpx.WriteError(
				w,
				http.StatusInternalServerError,
				"server error",
			)
		}

		return
	}

	httpx.WriteJSON(w, http.StatusCreated, importResponse)
}

func (handler *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, questionbank.ErrInvalidQuestionBankID):
		httpx.WriteError(w, http.StatusBadRequest, questionbank.ErrInvalidQuestionBankID.Error())
	case errors.Is(err, questionbank.ErrQuestionBankNotFound):
		httpx.WriteError(w, http.StatusNotFound, questionbank.ErrQuestionBankNotFound.Error())
	case errors.Is(err, importer.ErrMathNotSupported):
		httpx.WriteError(w, http.StatusUnprocessableEntity, importer.ErrMathNotSupported.Error())
	case errors.Is(err, importer.ErrMediaNotSupported):
		httpx.WriteError(w, http.StatusUnprocessableEntity, importer.ErrMediaNotSupported.Error())
	case errors.Is(err, ErrInvalidDocument):
		httpx.WriteError(w, http.StatusUnprocessableEntity, ErrInvalidDocument.Error())
	default:
		slog.Error("preview import", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "server error")
	}
}
