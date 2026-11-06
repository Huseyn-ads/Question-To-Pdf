package importer

// TextRun is a piece of paragraph text with the formatting that matters for
// answer detection. Formatting is already resolved through document
// defaults, paragraph styles and character styles.
type TextRun struct {
	Text string
	Bold bool
	// Highlight is the Word highlight colour name, for example "yellow".
	Highlight string
	// Color is a normalised hue class ("red", "blue", ...). Black, grey and
	// automatic colours are empty.
	Color     string
	Underline bool
	// Shading is the normalised hue class of the run background fill.
	Shading string
}

// ListKind describes how Word renders the automatic number of a paragraph.
type ListKind int

const (
	ListNone ListKind = iota
	ListNumber
	ListLetter
	ListOther
)

// Paragraph is one Word paragraph in document order.
type Paragraph struct {
	Runs []TextRun

	// ListLabel is the automatic number exactly as Word renders it, for
	// example "12." or "b)". It is empty for paragraphs outside lists.
	ListLabel string
	ListKind  ListKind
	// ListValue is the counter of the paragraph's own list level.
	ListValue int
	// ListKey identifies the list instance and level ("numId:ilvl").
	ListKey string

	// TableRow is a document-wide 1-based row number for paragraphs inside
	// table cells and 0 elsewhere.
	TableRow int
}

// Text returns the paragraph text without the automatic list label.
func (paragraph Paragraph) Text() string {
	total := 0
	for _, run := range paragraph.Runs {
		total += len(run.Text)
	}

	buffer := make([]byte, 0, total)
	for _, run := range paragraph.Runs {
		buffer = append(buffer, run.Text...)
	}

	return string(buffer)
}
