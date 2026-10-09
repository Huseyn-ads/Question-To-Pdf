package main

import (
	"fmt"
	"qtp/internal/importer"
)

func main() {
	paragraphs, err := importer.ExtractParagraphs("/Users/huseynshikhmammadli/VS Code/Question-To-Pdf/internal/importer/testdata/FƏLSƏFƏ TEST 300.docx")
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

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
