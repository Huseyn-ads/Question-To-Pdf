package importer

import (
	"fmt"
	"strings"
)

func ValidateQuestions(questions []QuestionDraft) []ValidationIssue {
	issues := make([]ValidationIssue, 0)

	if len(questions) == 0 {
		issues = append(issues, ValidationIssue{
			Field:   "questions",
			Code:    "questions_empty",
			Message: "document does not contain any questions",
		})

		return issues
	}

	seenNumbers := make(map[int]struct{}, len(questions))

	for _, question := range questions {
		if question.SourceNumber < 1 {
			issues = append(issues, ValidationIssue{
				SourceNumber: question.SourceNumber,
				Field:        "source_number",
				Code:         "invalid_question_number",
				Message:      "question number must be greater than zero",
			})
		} else {
			if _, exists := seenNumbers[question.SourceNumber]; exists {
				issues = append(issues, ValidationIssue{
					SourceNumber: question.SourceNumber,
					Field:        "source_number",
					Code:         "duplicate_question_number",
					Message:      "question number is duplicated",
				})
			} else {
				seenNumbers[question.SourceNumber] = struct{}{}
			}
		}

		if strings.TrimSpace(question.Text) == "" {
			issues = append(issues, ValidationIssue{
				SourceNumber: question.SourceNumber,
				Field:        "text",
				Code:         "question_text_required",
				Message:      "question text is required",
			})
		}

		if len(question.Options) < 2 {
			issues = append(issues, ValidationIssue{
				SourceNumber: question.SourceNumber,
				Field:        "options",
				Code:         "not_enough_options",
				Message:      "question must contain at least two options",
			})
		}

		if len(question.Options) > 10 {
			issues = append(issues, ValidationIssue{
				SourceNumber: question.SourceNumber,
				Field:        "options",
				Code:         "too_many_options",
				Message:      "question cannot contain more than ten options",
			})
		}

		seenOptions := make(map[string]struct{}, len(question.Options))
		seenLabels := make(map[string]struct{}, len(question.Options))
		correctOptionCount := 0

		for optionIndex, option := range question.Options {
			textField := fmt.Sprintf(
				"options[%d].text",
				optionIndex,
			)

			labelField := fmt.Sprintf(
				"options[%d].label",
				optionIndex,
			)

			normalizedText := strings.ToLower(
				strings.Join(
					strings.Fields(option.Text),
					" ",
				),
			)

			if normalizedText == "" {
				issues = append(issues, ValidationIssue{
					SourceNumber: question.SourceNumber,
					Field:        textField,
					Code:         "option_text_required",
					Message:      "option text is required",
				})
			} else {
				if _, exists := seenOptions[normalizedText]; exists {
					issues = append(issues, ValidationIssue{
						SourceNumber: question.SourceNumber,
						Field:        textField,
						Code:         "duplicate_option_text",
						Message:      "option text is duplicated",
					})
				} else {
					seenOptions[normalizedText] = struct{}{}
				}
			}

			normalizedLabel := strings.ToUpper(
				strings.TrimSpace(option.Label),
			)

			if normalizedLabel == "" {
				issues = append(issues, ValidationIssue{
					SourceNumber: question.SourceNumber,
					Field:        labelField,
					Code:         "option_label_required",
					Message:      "option label is required",
				})
			} else {
				if _, exists := seenLabels[normalizedLabel]; exists {
					issues = append(issues, ValidationIssue{
						SourceNumber: question.SourceNumber,
						Field:        labelField,
						Code:         "duplicate_option_label",
						Message:      "option label is duplicated",
					})
				} else {
					seenLabels[normalizedLabel] = struct{}{}
				}
			}

			if option.IsCorrect {
				correctOptionCount++
			}
		}

		if correctOptionCount != 1 {
			issues = append(issues, ValidationIssue{
				SourceNumber: question.SourceNumber,
				Field:        "options",
				Code:         "exactly_one_correct_option",
				Message: fmt.Sprintf(
					"question must contain exactly one correct option, found %d",
					correctOptionCount,
				),
			})
		}
	}

	return issues
}
