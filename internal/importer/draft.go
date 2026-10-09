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
	QuestionsCount int             `json:"questions_count"`
	Warnings       []string        `json:"warnings"`
	Questions      []QuestionDraft `json:"questions"`
}
