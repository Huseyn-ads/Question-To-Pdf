package questionimport

import (
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"qtp/internal/importer"
	"qtp/internal/questionbank"
)

type forbiddenReader struct{ t *testing.T }

func (r forbiddenReader) Read([]byte) (int, error) {
	r.t.Fatal("unauthorized request body was read")
	return 0, io.EOF
}

func TestUnauthorizedImportDoesNotReadBody(t *testing.T) {
	h := NewHandler(nil)
	for _, method := range []func(*httptest.ResponseRecorder){
		func(w *httptest.ResponseRecorder) { h.Preview(w, httptest.NewRequest("POST", "/", forbiddenReader{t})) },
		func(w *httptest.ResponseRecorder) { h.Import(w, httptest.NewRequest("POST", "/", forbiddenReader{t})) },
	} {
		w := httptest.NewRecorder()
		method(w)
		if w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
}

func TestImportErrorsDoNotExposeInternalPaths(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{fmt.Errorf("extract /private/document.docx: %w", ErrInvalidDocument), 422},
		{fmt.Errorf("extract /private/document.docx: %w", importer.ErrMathNotSupported), 422},
		{fmt.Errorf("extract /private/document.docx: %w", importer.ErrMediaNotSupported), 422},
		{fmt.Errorf("database SQL /private: %w", questionbank.ErrQuestionBankNotFound), 404},
		{fmt.Errorf("database SQL /private: %w", questionbank.ErrInvalidQuestionBankID), 400},
		{errors.New("SQL /private/document.docx"), 500},
	} {
		w := httptest.NewRecorder()
		(&Handler{}).writeError(w, tc.err)
		if w.Code != tc.status {
			t.Fatal(w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "/private") || strings.Contains(w.Body.String(), "SQL") {
			t.Fatal("internal data exposed")
		}
	}
}
