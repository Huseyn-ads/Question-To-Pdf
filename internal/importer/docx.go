package importer

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const maxXMLSize = 20 * 1024 * 1024

var wordNamespaces = map[string]bool{
	"http://schemas.openxmlformats.org/wordprocessingml/2006/main": true,
	"http://purl.oclc.org/ooxml/wordprocessingml/main":             true,
}

var mathNamespaces = map[string]bool{
	"http://schemas.openxmlformats.org/officeDocument/2006/math": true,
	"http://purl.oclc.org/ooxml/officeDocument/math":             true,
}

func ExtractParagraphs(
	filePath string,
) ([]Paragraph, error) {
	return ExtractParagraphsContext(context.Background(), filePath)
}

func openPart(archive *zip.ReadCloser, name string) (io.ReadCloser, *io.LimitedReader, error) {
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		if file.UncompressedSize64 > uint64(maxXMLSize) {
			return nil, nil, fmt.Errorf("%s exceeds 20 MiB", name)
		}
		reader, err := file.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("open %s: %w", name, err)
		}
		limited := &io.LimitedReader{R: reader, N: maxXMLSize + 1}
		return reader, limited, nil
	}
	return nil, nil, nil
}

func decodePart(archive *zip.ReadCloser, name string, target any) (bool, error) {
	closer, reader, err := openPart(archive, name)
	if err != nil || closer == nil {
		return false, err
	}
	defer closer.Close()

	if err := xml.NewDecoder(reader).Decode(target); err != nil {
		if reader.N == 0 {
			return false, fmt.Errorf("%s exceeds 20 MiB", name)
		}
		return false, fmt.Errorf("read %s: %w", name, err)
	}
	if reader.N == 0 {
		return false, fmt.Errorf("%s exceeds 20 MiB", name)
	}
	return true, nil
}

func attributeValue(element xml.StartElement, local string) (string, bool) {
	for _, attribute := range element.Attr {
		if attribute.Name.Local == local &&
			(attribute.Name.Space == "" || wordNamespaces[attribute.Name.Space]) {
			return attribute.Value, true
		}
	}
	return "", false
}

func ExtractParagraphsContext(ctx context.Context, filePath string) ([]Paragraph, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	archive, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf(
			"open DOCX archive: %w",
			err,
		)
	}
	defer archive.Close()

	var styles xmlStyles
	hasStyles, err := decodePart(archive, "word/styles.xml", &styles)
	if err != nil {
		return nil, err
	}
	sheet := newStyleSheet(nil)
	if hasStyles {
		sheet = newStyleSheet(&styles)
	}

	var numbering xmlNumbering
	hasNumbering, err := decodePart(archive, "word/numbering.xml", &numbering)
	if err != nil {
		return nil, err
	}
	lists := newNumberingState(nil, sheet)
	if hasNumbering {
		lists = newNumberingState(&numbering, sheet)
	}

	closer, limitedReader, err := openPart(archive, "word/document.xml")
	if err != nil {
		return nil, err
	}
	if closer == nil {
		return nil, errors.New(
			"word/document.xml not found",
		)
	}
	defer closer.Close()

	reader := &documentReader{
		decoder: xml.NewDecoder(limitedReader),
		styles:  sheet,
		lists:   lists,
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		token, err := reader.decoder.Token()

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

		if err := reader.handle(token); err != nil {
			return nil, err
		}
	}

	return reader.paragraphs, nil
}

type documentReader struct {
	decoder *xml.Decoder
	styles  *styleSheet
	lists   *numberingState

	paragraphs []Paragraph

	paragraph      *Paragraph
	paragraphStyle string
	numID          *string
	ilvl           int
	inParagraphPr  bool

	run          *TextRun
	runFormat    runFormat
	runStyle     string
	inRunPr      bool
	tableDepth   int
	tableRow     int
	tableRowSeen int
}

func (reader *documentReader) skip(what string) error {
	if err := reader.decoder.Skip(); err != nil {
		return fmt.Errorf("skip %s: %w", what, err)
	}
	return nil
}

func (reader *documentReader) handle(token xml.Token) error {
	switch element := token.(type) {
	case xml.StartElement:
		if mathNamespaces[element.Name.Space] {
			return fmt.Errorf(
				"%w: element %q",
				ErrMathNotSupported,
				element.Name.Local,
			)
		}

		if !wordNamespaces[element.Name.Space] {
			return nil
		}

		return reader.start(element)

	case xml.EndElement:
		if !wordNamespaces[element.Name.Space] {
			return nil
		}

		reader.end(element)
	}

	return nil
}

func (reader *documentReader) start(element xml.StartElement) error {
	switch element.Name.Local {
	case "tbl":
		reader.tableDepth++
	case "tr":
		if reader.tableDepth > 0 {
			reader.tableRowSeen++
			reader.tableRow = reader.tableRowSeen
		}

	case "p":
		if reader.paragraph != nil {
			return errors.New(
				"nested paragraphs are not supported",
			)
		}

		reader.paragraph = &Paragraph{Runs: make([]TextRun, 0)}
		if reader.tableDepth > 0 {
			reader.paragraph.TableRow = reader.tableRow
		}
		reader.paragraphStyle = ""
		reader.numID = nil
		reader.ilvl = 0

	case "pPr":
		if reader.paragraph != nil && reader.run == nil {
			reader.inParagraphPr = true
		}

	case "pStyle":
		if reader.inParagraphPr {
			reader.paragraphStyle, _ = attributeValue(element, "val")
		}

	case "numId":
		if reader.inParagraphPr {
			value, _ := attributeValue(element, "val")
			reader.numID = &value
		}

	case "ilvl":
		if reader.inParagraphPr {
			value, _ := attributeValue(element, "val")
			if level, err := strconv.Atoi(value); err == nil {
				reader.ilvl = level
			}
		}

	case "pPrChange", "rPrChange", "numberingChange":
		// Formatting history describes the previous state of the
		// document, not what the reader sees.
		return reader.skip("formatting history")

	case "del", "moveFrom":
		// Deleted text of tracked changes is not part of the document.
		return reader.skip("deleted text")

	case "r":
		if reader.paragraph != nil {
			reader.run = &TextRun{}
			reader.runFormat = runFormat{}
			reader.runStyle = ""
		}

	case "rPr":
		if reader.run != nil {
			reader.inRunPr = true
		} else if reader.inParagraphPr {
			// Paragraph mark properties do not format the text.
			return reader.skip("paragraph mark properties")
		}

	case "rStyle":
		if reader.inRunPr {
			reader.runStyle, _ = attributeValue(element, "val")
		}

	case "b", "highlight", "color", "u", "shd", "vanish":
		if !reader.inRunPr {
			return nil
		}
		return reader.runProperty(element)

	case "t":
		if reader.run == nil {
			return nil
		}

		var text string
		if err := reader.decoder.DecodeElement(&text, &element); err != nil {
			return fmt.Errorf(
				"decode Word text: %w",
				err,
			)
		}
		reader.run.Text += text

	case "tab", "ptab":
		if reader.run != nil && !reader.inRunPr {
			reader.run.Text += "\t"
		}

	case "br", "cr":
		if reader.run != nil {
			reader.run.Text += "\n"
		}

	case "noBreakHyphen":
		if reader.run != nil {
			reader.run.Text += "-"
		}

	case "drawing", "pict", "object":
		return fmt.Errorf(
			"%w: element %q",
			ErrMediaNotSupported,
			element.Name.Local,
		)
	}

	return nil
}

func (reader *documentReader) runProperty(element xml.StartElement) error {
	value, hasValue := attributeValue(element, "val")
	property := &xmlRunProperties{}
	switch element.Name.Local {
	case "b":
		property.Bold = &xmlValue{Val: value}
	case "highlight":
		property.Highlight = &xmlValue{Val: value}
	case "color":
		property.Color = &xmlValue{Val: value}
	case "u":
		if !hasValue {
			value = "single"
		}
		property.Underline = &xmlValue{Val: value}
	case "shd":
		fill, _ := attributeValue(element, "fill")
		property.Shading = &xmlShading{Fill: fill, Val: value}
	case "vanish":
		property.Vanish = &xmlValue{Val: value}
	}
	reader.runFormat = formatFromXML(property).over(reader.runFormat)
	return nil
}

func (reader *documentReader) end(element xml.EndElement) {
	switch element.Name.Local {
	case "tbl":
		if reader.tableDepth > 0 {
			reader.tableDepth--
		}
		if reader.tableDepth == 0 {
			reader.tableRow = 0
		}

	case "pPr":
		reader.inParagraphPr = false

	case "rPr":
		reader.inRunPr = false

	case "r":
		if reader.paragraph != nil && reader.run != nil {
			format := reader.runFormat.
				over(reader.styles.runFormat(reader.runStyle)).
				over(reader.styles.runFormat(reader.paragraphStyleID())).
				over(reader.styles.defaults)
			if format.vanish == nil || !*format.vanish {
				format.apply(reader.run)
				if reader.run.Text != "" {
					reader.paragraph.Runs = append(reader.paragraph.Runs, *reader.run)
				}
			}
		}

		reader.run = nil
		reader.inRunPr = false

	case "p":
		if reader.paragraph != nil {
			reader.finishParagraph()
			reader.paragraphs = append(reader.paragraphs, *reader.paragraph)
		}

		reader.paragraph = nil
		reader.run = nil
		reader.inParagraphPr = false
		reader.inRunPr = false
	}
}

func (reader *documentReader) paragraphStyleID() string {
	if reader.paragraphStyle != "" {
		return reader.paragraphStyle
	}
	return reader.styles.defaultParagraphID
}

func (reader *documentReader) finishParagraph() {
	numID := ""
	ilvl := reader.ilvl
	if reader.numID != nil {
		numID = *reader.numID
	} else if inherited := reader.styles.numbering(reader.paragraphStyleID()); inherited != nil {
		numID = inherited.NumID.Val
		if inherited.Ilvl != nil {
			if level, err := strconv.Atoi(inherited.Ilvl.Val); err == nil {
				ilvl = level
			}
		}
	}

	label, kind, value, ok := reader.lists.next(numID, ilvl)
	if !ok {
		return
	}

	reader.paragraph.ListLabel = label
	reader.paragraph.ListKind = kind
	reader.paragraph.ListValue = value
	reader.paragraph.ListKey = numID + ":" + strconv.Itoa(ilvl)
}
