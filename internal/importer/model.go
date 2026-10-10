package importer

type TextRun struct {
	Text string
	Bold bool
}

type Paragraph struct {
	Runs []TextRun
}
