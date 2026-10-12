package questionimport

import "qtp/internal/importer"

type ImportRequest struct {
	Questions []importer.QuestionDraft `json:"questions"`
}

type ImportResponse struct {
	ImportedQuestions int `json:"imported_questions"`
	ImportedOptions   int `json:"imported_options"`
}
