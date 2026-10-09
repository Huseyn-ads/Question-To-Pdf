package importer

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	numberedLinePattern = regexp.MustCompile(
		`^[\t\p{Zs}]*(\d+)[.)][\t\p{Zs}]*(.+?)[\t\p{Zs}]*$`,
	)

	optionStartPattern = regexp.MustCompile(
		`(?i)(^|[\t\r\n\p{Zs};;]+)([A-E])[.)][\t\r\n\p{Zs}]*`,
	)

	trailingDifficultyPattern = regexp.MustCompile(
		`(?s)^(.*?)[\t\p{Zs}]{4,}[AOÇaoç]$`,
	)
)

type styledParagraph struct {
	Text   string
	BoldAt []bool
}

func parseNumberedLine(text string) (int, string, bool) {
	matches := numberedLinePattern.FindStringSubmatch(text)

	if len(matches) != 3 {
		return 0, "", false
	}

	number, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", false
	}

	content := strings.TrimSpace(matches[2])

	return number, content, true
}

func removeTrailingDifficulty(text string) string {
	matches := trailingDifficultyPattern.FindStringSubmatch(text)

	if len(matches) != 2 {
		return strings.TrimSpace(text)
	}

	return strings.TrimSpace(matches[1])
}

func flattenParagraph(
	paragraph Paragraph,
) styledParagraph {
	totalLength := 0

	for _, run := range paragraph.Runs {
		totalLength += len(run.Text)
	}

	var text strings.Builder
	text.Grow(totalLength)

	boldAt := make([]bool, 0, totalLength)

	for _, run := range paragraph.Runs {
		text.WriteString(run.Text)

		for i := 0; i < len(run.Text); i++ {
			boldAt = append(boldAt, run.Bold)
		}
	}

	return styledParagraph{
		Text:   text.String(),
		BoldAt: boldAt,
	}
}

func isRangeMostlyBold(
	styled styledParagraph,
	start int,
	end int,
) bool {
	if start < 0 ||
		end > len(styled.Text) ||
		end > len(styled.BoldAt) ||
		start >= end {
		return false
	}

	totalCount := 0
	boldCount := 0

	for index := start; index < end; {
		symbol, size := utf8.DecodeRuneInString(
			styled.Text[index:end],
		)

		if !unicode.IsSpace(symbol) {
			totalCount++

			if styled.BoldAt[index] {
				boldCount++
			}
		}

		index += size
	}

	if totalCount == 0 {
		return false
	}

	return boldCount*2 >= totalCount
}

func parseOptions(
	paragraph Paragraph,
) []OptionDraft {
	styled := flattenParagraph(paragraph)

	matches := optionStartPattern.FindAllStringSubmatchIndex(
		styled.Text,
		-1,
	)

	if len(matches) == 0 {
		return nil
	}

	options := make(
		[]OptionDraft,
		0,
		len(matches),
	)

	for index, match := range matches {
		labelStart := match[4]
		labelEnd := match[5]

		contentStart := match[1]
		contentEnd := len(styled.Text)

		if index+1 < len(matches) {
			contentEnd = matches[index+1][0]
		}

		label := strings.ToUpper(
			styled.Text[labelStart:labelEnd],
		)

		rawText := styled.Text[contentStart:contentEnd]
		cleanText := removeTrailingDifficulty(rawText)

		boldStart := contentStart
		boldEnd := contentEnd

		if cleanText != "" {
			offset := strings.Index(rawText, cleanText)

			if offset >= 0 {
				boldStart = contentStart + offset
				boldEnd = boldStart + len(cleanText)
			}
		}

		options = append(options, OptionDraft{
			Label: label,
			Text:  cleanText,
			IsCorrect: isRangeMostlyBold(
				styled,
				boldStart,
				boldEnd,
			),
		})
	}

	return options
}

func textBeforeFirstOption(
	paragraph Paragraph,
) string {
	styled := flattenParagraph(paragraph)

	matches := optionStartPattern.FindAllStringSubmatchIndex(
		styled.Text,
		-1,
	)

	if len(matches) == 0 {
		return strings.TrimSpace(styled.Text)
	}

	return strings.TrimSpace(
		styled.Text[:matches[0][0]],
	)
}

func ParseQuestions(
	paragraphs []Paragraph,
) []QuestionDraft {
	questions := make([]QuestionDraft, 0)
	var currentQuestion *QuestionDraft

	for _, paragraph := range paragraphs {
		textBeforeOptions :=
			textBeforeFirstOption(paragraph)

		options := parseOptions(paragraph)

		number, questionText, isNumberedLine :=
			parseNumberedLine(textBeforeOptions)

		if isNumberedLine {
			if currentQuestion != nil &&
				len(currentQuestion.Options) > 0 {
				questions = append(
					questions,
					*currentQuestion,
				)
			}

			currentQuestion = &QuestionDraft{
				SourceNumber: number,
				Text:         questionText,
				Options:      make([]OptionDraft, 0, 5),
			}
		} else if currentQuestion != nil &&
			len(currentQuestion.Options) == 0 &&
			textBeforeOptions != "" {
			currentQuestion.Text = strings.TrimSpace(
				currentQuestion.Text +
					" " +
					textBeforeOptions,
			)
		}

		if currentQuestion != nil &&
			len(options) > 0 {
			currentQuestion.Options = append(
				currentQuestion.Options,
				options...,
			)
		}
	}

	if currentQuestion != nil &&
		len(currentQuestion.Options) > 0 {
		questions = append(
			questions,
			*currentQuestion,
		)
	}

	return questions
}
