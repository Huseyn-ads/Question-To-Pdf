package importer

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

func TextBeforeFirstOption(paragraph Paragraph) string {
	styled := FlattenParagraph(paragraph)

	matches := OptionStartPattern.FindAllStringSubmatchIndex(
		styled.Text,
		-1,
	)

	if len(matches) == 0 {
		return strings.TrimSpace(styled.Text)
	}

	firstOptionStart := matches[0][0]

	return strings.TrimSpace(
		styled.Text[:firstOptionStart],
	)
}

func ParseQuestions(
	paragraphs []Paragraph,
) []ParsedQuestion {
	questions := make([]ParsedQuestion, 0)
	var currentQuestion *ParsedQuestion

	for _, paragraph := range paragraphs {
		textBeforeOptions :=
			TextBeforeFirstOption(paragraph)

		options := ParseOptions(paragraph)

		number, questionText, isNumberedLine :=
			ParseNumberedLine(textBeforeOptions)

		if isNumberedLine {
			if currentQuestion != nil &&
				len(currentQuestion.Options) > 0 {
				questions = append(
					questions,
					*currentQuestion,
				)
			}

			currentQuestion = &ParsedQuestion{
				SourceNumber: number,
				Text:         questionText,
				Options:      make([]ParsedOption, 0, 5),
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

func ExtractParagraphs(filePath string) ([]Paragraph, error) {
	const wordNamespace = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	const mathNamespace = "http://schemas.openxmlformats.org/officeDocument/2006/math"
	const maxXMLSize = 20 * 1024 * 1024

	// 1. Открываем архив.
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open DOCX archive: %w", err)
	}
	defer archive.Close()

	// 2. Находим основной файл документа.
	var documentFile *zip.File

	for _, file := range archive.File {
		if file.Name == "word/document.xml" {
			documentFile = file
			break
		}
	}

	if documentFile == nil {
		return nil, errors.New("word/document.xml not found")
	}

	// 3. Открываем содержимое найденного файла.
	xmlReader, err := documentFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open document XML: %w", err)
	}
	defer xmlReader.Close()

	// Ограничиваем объём распакованного XML.
	limitedReader := &io.LimitedReader{
		R: xmlReader,
		N: maxXMLSize + 1,
	}

	decoder := xml.NewDecoder(limitedReader)

	paragraphs := make([]Paragraph, 0)

	var currentParagraph *Paragraph
	var currentRun *TextRun

	// 4. Последовательно читаем XML.
	for {
		token, err := decoder.Token()

		if limitedReader.N == 0 {
			return nil, errors.New("document XML exceeds 20 MiB")
		}

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read document XML: %w", err)
		}

		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Space == mathNamespace {
				if currentRun != nil {
					currentRun.Text += " [FORMULA] "
				} else if currentParagraph != nil {
					currentParagraph.Runs = append(
						currentParagraph.Runs,
						TextRun{
							Text: " [FORMULA] ",
						},
					)
				}

				if err := decoder.Skip(); err != nil {
					return nil, fmt.Errorf(
						"skip Word formula: %w",
						err,
					)
				}

				continue
			}
			// if element.Name.Space == mathNamespace {
			// 	return nil, errors.New(
			// 		"Word math is not supported by this extractor yet",
			// 	)
			// }

			if element.Name.Space != wordNamespace {
				continue
			}

			switch element.Name.Local {
			case "p":
				if currentParagraph != nil {
					return nil, errors.New(
						"nested paragraphs are not supported yet",
					)
				}

				currentParagraph = &Paragraph{
					Runs: make([]TextRun, 0),
				}

			case "r":
				if currentParagraph != nil {
					currentRun = &TextRun{}
				}

			case "t":
				if currentRun == nil {
					continue
				}

				var text string

				if err := decoder.DecodeElement(&text, &element); err != nil {
					return nil, fmt.Errorf("decode Word text: %w", err)
				}

				currentRun.Text += text

			case "b":
				if currentRun == nil {
					continue
				}

				currentRun.Bold = true

				for _, attribute := range element.Attr {
					if attribute.Name.Space == wordNamespace &&
						attribute.Name.Local == "val" {
						switch attribute.Value {
						case "false", "0", "off":
							currentRun.Bold = false
						}
					}
				}

			case "tab":
				if currentRun != nil {
					currentRun.Text += "\t"
				}

			case "br", "cr":
				if currentRun != nil {
					currentRun.Text += "\n"
				}

			case "rPrChange":
				// Пропускаем историю изменения оформления.
				if err := decoder.Skip(); err != nil {
					return nil, fmt.Errorf("skip formatting history: %w", err)
				}

			case "drawing", "pict", "object":
				placeholder := " [MEDIA] "

				if element.Name.Local == "object" {
					placeholder = " [FORMULA_OBJECT] "
				}

				if currentRun != nil {
					currentRun.Text += placeholder
				} else if currentParagraph != nil {
					currentParagraph.Runs = append(
						currentParagraph.Runs,
						TextRun{
							Text: placeholder,
						},
					)
				}

				if err := decoder.Skip(); err != nil {
					return nil, fmt.Errorf(
						"skip Word element %q: %w",
						element.Name.Local,
						err,
					)
				}
			}

		case xml.EndElement:
			if element.Name.Space != wordNamespace {
				continue
			}

			switch element.Name.Local {
			case "r":
				if currentParagraph != nil && currentRun != nil {
					currentParagraph.Runs = append(
						currentParagraph.Runs,
						*currentRun,
					)
				}

				currentRun = nil

			case "p":
				if currentParagraph != nil {
					paragraphs = append(paragraphs, *currentParagraph)
				}

				currentParagraph = nil
				currentRun = nil
			}
		}
	}

	return paragraphs, nil
}

func (p Paragraph) Text() string {
	pustaya := ""

	for _, run := range p.Runs {
		pustaya = pustaya + run.Text
	}

	return pustaya
}
