package questionimport

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"qtp/internal/importer"
)

func TestAnswerModeFromRequest(t *testing.T) {
	newRequest := func(target string, field string) *http.Request {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if field != "" {
			if err := writer.WriteField("answer_mode", field); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest("POST", target, &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		if err := request.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		return request
	}

	for _, tc := range []struct {
		target string
		field  string
		mode   string
		fails  bool
	}{
		{"/", "", importer.AnswerModeAuto, false},
		{"/", "all_a", importer.AnswerModeAllA, false},
		{"/?answer_mode=all_a", "", importer.AnswerModeAllA, false},
		{"/?answer_mode=all_a", "all_a", importer.AnswerModeAllA, false},
		{"/?answer_mode=auto", "all_a", "", true},
		{"/", "all_b", "", true},
	} {
		mode, err := answerModeFromRequest(newRequest(tc.target, tc.field))
		if (err != nil) != tc.fails || mode != tc.mode {
			t.Errorf("%s field=%q: mode %q, error %v", tc.target, tc.field, mode, err)
		}
	}
}
