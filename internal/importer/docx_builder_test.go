package importer

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helpers that build small DOCX files with the WordprocessingML features
// found in real exam documents.

const (
	propBold      = `<w:b/>`
	propNotBold   = `<w:b w:val="0"/>`
	propHighlight = `<w:highlight w:val="yellow"/>`
	propRed       = `<w:color w:val="FF0000"/>`
	propDarkRed   = `<w:color w:val="EE0000"/>`
	propHidden    = `<w:vanish/>`
	propStrong    = `<w:rStyle w:val="Strong"/>`
)

func run(text string, properties ...string) string {
	escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
	parts := strings.Split(escaped, "\n")
	var body strings.Builder
	for index, part := range parts {
		if index > 0 {
			body.WriteString("<w:br/>")
		}
		body.WriteString(`<w:t xml:space="preserve">` + part + `</w:t>`)
	}
	return "<w:r><w:rPr>" + strings.Join(properties, "") + "</w:rPr>" + body.String() + "</w:r>"
}

func para(runs ...string) string {
	return "<w:p>" + strings.Join(runs, "") + "</w:p>"
}

func listPara(numID, level int, runs ...string) string {
	return fmt.Sprintf(
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="%d"/><w:numId w:val="%d"/></w:numPr></w:pPr>%s</w:p>`,
		level, numID, strings.Join(runs, ""),
	)
}

type listLevel struct {
	format string
	text   string
	start  int
}

// list defines w:num numID with its own abstract list.
type list struct {
	numID  int
	levels []listLevel
}

func numberingXML(lists ...list) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	for _, definition := range lists {
		fmt.Fprintf(&builder, `<w:abstractNum w:abstractNumId="%d">`, definition.numID)
		for index, level := range definition.levels {
			start := level.start
			if start == 0 {
				start = 1
			}
			fmt.Fprintf(&builder,
				`<w:lvl w:ilvl="%d"><w:start w:val="%d"/><w:numFmt w:val="%s"/><w:lvlText w:val="%s"/></w:lvl>`,
				index, start, level.format, level.text)
		}
		builder.WriteString(`</w:abstractNum>`)
	}
	for _, definition := range lists {
		fmt.Fprintf(&builder, `<w:num w:numId="%d"><w:abstractNumId w:val="%d"/></w:num>`,
			definition.numID, definition.numID)
	}
	builder.WriteString(`</w:numbering>`)
	return builder.String()
}

var (
	decimalDot   = listLevel{format: "decimal", text: "%1."}
	decimalParen = listLevel{format: "decimal", text: "%1)"}
	upperDot     = listLevel{format: "upperLetter", text: "%1."}
	upperParen   = listLevel{format: "upperLetter", text: "%1)"}
	lowerParen   = listLevel{format: "lowerLetter", text: "%1)"}
	lowerDot     = listLevel{format: "lowerLetter", text: "%1."}
)

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
	`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>` +
	`<w:style w:type="character" w:styleId="Strong"><w:name w:val="Strong"/><w:rPr><w:b/></w:rPr></w:style>` +
	`</w:styles>`

func writeDocx(t *testing.T, body string, numbering string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.docx")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	parts := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
			body + `</w:body></w:document>`,
		"word/styles.xml": stylesXML,
	}
	if numbering != "" {
		parts["word/numbering.xml"] = numbering
	}
	for name, content := range parts {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func parseDocx(t *testing.T, body, numbering string, mode string) ParseResult {
	t.Helper()
	paragraphs, err := ExtractParagraphs(writeDocx(t, body, numbering))
	if err != nil {
		t.Fatal(err)
	}
	return Parse(paragraphs, ParseOptions{AnswerMode: mode})
}

// expected question: text, options as "L:text" with a leading "*" for the
// correct option, for example "*B:Paris".
type want struct {
	number  int
	text    string
	options []string
}

func checkQuestions(t *testing.T, got []QuestionDraft, expected []want) {
	t.Helper()
	if len(got) != len(expected) {
		for _, question := range got {
			t.Logf("got %d %q %v", question.SourceNumber, question.Text, question.Options)
		}
		t.Fatalf("questions: got %d, want %d", len(got), len(expected))
	}
	for index, question := range got {
		w := expected[index]
		if w.number != 0 && question.SourceNumber != w.number {
			t.Errorf("question %d: source_number %d, want %d", index+1, question.SourceNumber, w.number)
		}
		if question.Text != w.text {
			t.Errorf("question %d: text %q, want %q", index+1, question.Text, w.text)
		}
		options := make([]string, 0, len(question.Options))
		for _, option := range question.Options {
			mark := ""
			if option.IsCorrect {
				mark = "*"
			}
			options = append(options, mark+option.Label+":"+option.Text)
		}
		if strings.Join(options, " | ") != strings.Join(w.options, " | ") {
			t.Errorf("question %d options:\n got  %v\n want %v", index+1, options, w.options)
		}
	}
}

func issueCodes(issues []ValidationIssue) map[string][]int {
	result := make(map[string][]int)
	for _, issue := range issues {
		result[issue.Code] = append(result[issue.Code], issue.SourceNumber)
	}
	return result
}

func checkIssues(t *testing.T, issues []ValidationIssue, expected map[string][]int) {
	t.Helper()
	got := issueCodes(issues)
	if fmt.Sprint(got) != fmt.Sprint(expected) {
		for _, issue := range issues {
			t.Logf("issue %+v", issue)
		}
		t.Fatalf("issues:\n got  %v\n want %v", got, expected)
	}
}
