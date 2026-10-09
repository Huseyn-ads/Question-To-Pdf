package importer

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

func ExtractParagraphs(
	filePath string,
) ([]Paragraph, error) {
	const (
		wordNamespace = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
		mathNamespace = "http://schemas.openxmlformats.org/officeDocument/2006/math"
		maxXMLSize    = 20 * 1024 * 1024
	)

	archive, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf(
			"open DOCX archive: %w",
			err,
		)
	}
	defer archive.Close()

	var documentFile *zip.File

	for _, file := range archive.File {
		if file.Name == "word/document.xml" {
			documentFile = file
			break
		}
	}

	if documentFile == nil {
		return nil, errors.New(
			"word/document.xml not found",
		)
	}

	if documentFile.UncompressedSize64 >
		uint64(maxXMLSize) {
		return nil, errors.New(
			"document XML exceeds 20 MiB",
		)
	}

	xmlReader, err := documentFile.Open()
	if err != nil {
		return nil, fmt.Errorf(
			"open document XML: %w",
			err,
		)
	}
	defer xmlReader.Close()

	limitedReader := &io.LimitedReader{
		R: xmlReader,
		N: maxXMLSize + 1,
	}

	decoder := xml.NewDecoder(limitedReader)

	paragraphs := make([]Paragraph, 0)

	var currentParagraph *Paragraph
	var currentRun *TextRun

	for {
		token, err := decoder.Token()

		if limitedReader.N == 0 {
			return nil, errors.New(
				"document XML exceeds 20 MiB",
			)
		}

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf(
				"read document XML: %w",
				err,
			)
		}

		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Space == mathNamespace {
				return nil, fmt.Errorf(
					"%w: element %q",
					ErrMathNotSupported,
					element.Name.Local,
				)
			}

			if element.Name.Space != wordNamespace {
				continue
			}

			switch element.Name.Local {
			case "p":
				if currentParagraph != nil {
					return nil, errors.New(
						"nested paragraphs are not supported",
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

				if err := decoder.DecodeElement(
					&text,
					&element,
				); err != nil {
					return nil, fmt.Errorf(
						"decode Word text: %w",
						err,
					)
				}

				currentRun.Text += text

			case "b":
				if currentRun == nil {
					continue
				}

				currentRun.Bold = true

				for _, attribute := range element.Attr {
					if attribute.Name.Space != wordNamespace ||
						attribute.Name.Local != "val" {
						continue
					}

					switch attribute.Value {
					case "false", "0", "off":
						currentRun.Bold = false
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
				if err := decoder.Skip(); err != nil {
					return nil, fmt.Errorf(
						"skip formatting history: %w",
						err,
					)
				}

			case "drawing", "pict", "object":
				return nil, fmt.Errorf(
					"%w: element %q",
					ErrMediaNotSupported,
					element.Name.Local,
				)
			}

		case xml.EndElement:
			if element.Name.Space != wordNamespace {
				continue
			}

			switch element.Name.Local {
			case "r":
				if currentParagraph != nil &&
					currentRun != nil {
					currentParagraph.Runs = append(
						currentParagraph.Runs,
						*currentRun,
					)
				}

				currentRun = nil

			case "p":
				if currentParagraph != nil {
					paragraphs = append(
						paragraphs,
						*currentParagraph,
					)
				}

				currentParagraph = nil
				currentRun = nil
			}
		}
	}

	return paragraphs, nil
}
