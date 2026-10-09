package main

import (
	"fmt"
	"qtp/internal/importer"
	"strings"
)

func main() {
	paragraphs, err := importer.ExtractParagraphs("/Users/huseynshikhmammadli/VS Code/Question-To-Pdf/internal/importer/testdata/FƏLSƏFƏ TEST 300.docx")
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	fmt.Println("Абзацев:", len(paragraphs))

	// for i, paragraph := range paragraphs {
	// 	if i >= 10 {
	// 		break
	// 	}

	// 	fmt.Printf("Абзац %d:\n", i+1)

	// 	for _, run := range paragraph.Runs {
	// 		fmt.Printf("  bold=%t text=%q\n", run.Bold, run.Text)
	// 	}
	// }

	// fmt.Println(strings.TrimSpace(paragraphs[4].Text()))

	// number, content, ok := importer.ParseNumberedLine()

	// fmt.Println(number)
	// fmt.Println(content)
	// fmt.Println(ok)

	// for _, paragraph := range paragraphs[:20] {
	// 	text := importer.RemoveTrailingDifficulty(paragraph.Text())
	// 	label, content, ok := importer.ParseOptionLine(text)
	// 	if ok {
	// 		fmt.Printf("%s: %s\n", label, content)
	// 	}
	// }

	// for _, paragraph := range paragraphs {
	// 	text := importer.RemoveTrailingDifficulty(paragraph.Text())

	// 	label, content, ok := importer.ParseOptionLine(text)
	// 	if !ok {
	// 		continue
	// 	}

	// 	fmt.Printf(
	// 		"%s: %s | correct=%t\n",
	// 		label,
	// 		content,
	// 		importer.IsMostlyBold(paragraph),
	// 	)

	// 	styled := importer.FlattenParagraph(paragraph)

	// 	fmt.Println(len(styled.Text))
	// 	fmt.Println(len(styled.BoldAt))
	// 	fmt.Println(len(styled.Text) == len(styled.BoldAt))
	// }

	found := 0

	for _, paragraph := range paragraphs {
		options := importer.ParseOptions(paragraph)

		// if len(options) <= 1 {
		// 	continue
		// }

		for _, option := range options {
			fmt.Printf(
				"%s: %s | correct=%t\n",
				option.Label,
				option.Text,
				option.IsCorrect,
			)
		}

		fmt.Println()

		found++

	}

	totalOptions := 0
	correctOptions := 0
	emptyOptions := 0
	unparsedMarkers := 0

	for _, paragraph := range paragraphs {
		options := importer.ParseOptions(paragraph)

		for _, option := range options {
			totalOptions++

			if option.IsCorrect {
				correctOptions++
			}

			if strings.TrimSpace(option.Text) == "" {
				emptyOptions++
			}

			if importer.OptionStartPattern.MatchString(option.Text) {
				unparsedMarkers++
			}
		}
	}

	fmt.Println("Total options:", totalOptions)
	fmt.Println("Correct options:", correctOptions)
	fmt.Println("Empty options:", emptyOptions)
	fmt.Println("Unparsed markers:", unparsedMarkers)

	questions := importer.ParseQuestions(paragraphs)

	badOptionCount := 0
	badCorrectCount := 0
	badLabels := 0

	expectedLabels := []string{
		"A", "B", "C", "D", "E",
	}

	for _, question := range questions {
		if len(question.Options) != 5 {
			badOptionCount++

			fmt.Printf(
				"Question %d has %d options\n",
				question.SourceNumber,
				len(question.Options),
			)

			continue
		}

		correctCount := 0

		for index, option := range question.Options {
			if option.IsCorrect {
				correctCount++
			}

			if option.Label != expectedLabels[index] {
				badLabels++
			}
		}

		if correctCount != 1 {
			badCorrectCount++

			fmt.Printf(
				"Question %d has %d correct options\n",
				question.SourceNumber,
				correctCount,
			)
		}
	}

	fmt.Println("Questions:", len(questions))
	fmt.Println("Bad option count:", badOptionCount)
	fmt.Println("Bad correct count:", badCorrectCount)
	fmt.Println("Bad labels:", badLabels)
}
