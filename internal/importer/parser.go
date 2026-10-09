package importer

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var numberedLinePattern = regexp.MustCompile(
	`^[\t\p{Zs}]*(\d+)[.)][\t\p{Zs}]*(.+?)[\t\p{Zs}]*$`,
)
var optionLinePattern = regexp.MustCompile(`^([A-Ea-e])[.)]\s*(.+)$`)
var OptionStartPattern = regexp.MustCompile(`(?i)(^|[\t\r\n\p{Zs};;]+)([A-E])[.)][\t\r\n\p{Zs}]*`)

var trailingDifficultyPattern = regexp.MustCompile(`(?s)^(.*?)[\t\p{Zs}]{4,}[AOÇaoç]$`)

func ParseNumberedLine(text string) (int, string, bool) {
	matches := numberedLinePattern.FindStringSubmatch(text)

	if len(matches) != 3 {
		return 0, "", false
	}

	questionNumber, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", false
	}

	content := strings.TrimSpace(matches[2])

	return questionNumber, content, true
}

func ParseOptionLine(text string) (label string, content string, ok bool) {
	text = strings.TrimSpace(text)
	matches := optionLinePattern.FindStringSubmatch(text)

	if len(matches) != 3 {
		return "", "", false
	}

	label = strings.ToUpper(matches[1])

	content = strings.TrimSpace(matches[2])

	return label, content, true
}

func RemoveTrailingDifficulty(text string) string {
	matches := trailingDifficultyPattern.FindStringSubmatch(text)

	if len(matches) != 2 {
		return strings.TrimSpace(text)
	}

	return strings.TrimSpace(matches[1])
}

func IsMostlyBold(paragraph Paragraph) bool {
	totalCount := 0
	boldCount := 0

	for _, run := range paragraph.Runs {
		for _, symbol := range run.Text {
			if unicode.IsSpace(symbol) {
				continue
			}

			totalCount++

			if run.Bold {
				boldCount++
			}
		}
	}

	if totalCount == 0 {
		return false
	}

	return boldCount*2 >= totalCount
}

func FlattenParagraph(paragraph Paragraph) StyledParagraph {
	var text strings.Builder
	boldAt := make([]bool, 0)

	for _, run := range paragraph.Runs {
		text.WriteString(run.Text)

		for i := 0; i < len(run.Text); i++ {
			boldAt = append(boldAt, run.Bold)
		}
	}

	return StyledParagraph{
		Text:   text.String(),
		BoldAt: boldAt,
	}
}

func IsRangeMostlyBold(styled StyledParagraph, start int, end int) bool {
	if start < 0 ||
		end > len(styled.Text) ||
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

func ParseOptions(paragraph Paragraph) []ParsedOption {
	styled := FlattenParagraph(paragraph)

	matches := OptionStartPattern.FindAllStringSubmatchIndex(
		styled.Text,
		-1,
	)

	if len(matches) == 0 {
		return nil
	}

	options := make(
		[]ParsedOption,
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
		cleanText := RemoveTrailingDifficulty(rawText)

		boldStart := contentStart
		boldEnd := contentEnd

		if cleanText != "" {
			offset := strings.Index(rawText, cleanText)

			if offset >= 0 {
				boldStart = contentStart + offset
				boldEnd = boldStart + len(cleanText)
			}
		}

		options = append(options, ParsedOption{
			Label: label,
			Text:  cleanText,
			IsCorrect: IsRangeMostlyBold(
				styled,
				boldStart,
				boldEnd,
			),
		})
	}

	return options
}
