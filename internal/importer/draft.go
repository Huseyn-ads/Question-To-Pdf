package importer

type OptionDraft struct {
	Label     string `json:"label"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

type QuestionDraft struct {
	SourceNumber int           `json:"source_number"`
	Text         string        `json:"text"`
	Options      []OptionDraft `json:"options"`
}

type PreviewResult struct {
	QuestionsCount int               `json:"questions_count"`
	CanImport      bool              `json:"can_import"`
	Issues         []ValidationIssue `json:"issues"`
	Questions      []QuestionDraft   `json:"questions"`
}

type ValidationIssue struct {
	SourceNumber int    `json:"source_number,omitempty"`
	Field        string `json:"field,omitempty"`
	Code         string `json:"code"`
	Message      string `json:"message"`
}
