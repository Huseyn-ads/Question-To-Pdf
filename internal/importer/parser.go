package importer

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The parser works on paragraphs in document order and recognises the
// question layouts found in real exam documents:
//
//   - question and option numbers typed as text ("12.", "B)") or produced by
//     Word lists (the label exists only in numbering.xml);
//   - options one per paragraph, several per paragraph ("a) x  b) y"), split
//     by line breaks inside one paragraph, or starting in the question line;
//   - stems between the question line and the options (sentences, code,
//     numbered statements or matching tables);
//   - section headings, difficulty markers and weights around questions.
//
// A question starts at a numbered paragraph that is not part of a
// consecutive numbered run. The options of a question are its last group of
// letter options (or, without letters, a group of 3–10 numbered options
// starting at 1); everything between the question line and that group is
// the stem.

type labelKind int

const (
	labelNone labelKind = iota
	labelNumber
	labelLetter
)

const maxLetterValue = 8 // A–H

var (
	manualNumberPattern = regexp.MustCompile(
		`^[\s\p{Zs}]*(\d{1,4})[ \t\p{Zs}]*[.)]`,
	)

	manualLetterPattern = regexp.MustCompile(
		`^[\s\p{Zs}]*([A-Ha-h])(?:[ \t\p{Zs}]*\)|\.)`,
	)

	trailingDifficultyPattern = regexp.MustCompile(
		`(?s)^(.*?)[\t\p{Zs}]{4,}[AOÇaoç]$`,
	)

	// Difficulty or weight annotations written after an option, for example
	// "Sosial\t\t(çətin tipli)" or "Planlama   Çəki2".
	trailingMetadataPattern = regexp.MustCompile(
		`(?is)^(.*\S)(?:[\t ]*\(\s*(?:(?:asan|orta|çətin)(?:\s+tipli)?|çəki\s*\d+)\s*\)|(?:\t|[\t ]{2,})(?:(?:asan|orta|çətin)(?:\s+tipli)?|çəki\s*\d+))\s*$`,
	)

	parenDifficultyPattern = regexp.MustCompile(
		`[\s\p{Zs}]*\([\s\p{Zs}]*[aoçAOÇ][\s\p{Zs}]*\)[\s\p{Zs}]*$`,
	)

	letterDifficultyPattern = regexp.MustCompile(
		`(?:^|[\s\p{Zs}.…:?!;,])[AOÇ][\s\p{Zs}]*$`,
	)
)

type styledText struct {
	text  string
	runOf []int
	runs  []TextRun
}

func newStyledText(paragraph Paragraph) styledText {
	text := paragraph.Text()
	runOf := make([]int, 0, len(text))
	for index, run := range paragraph.Runs {
		for range len(run.Text) {
			runOf = append(runOf, index)
		}
	}
	return styledText{text: text, runOf: runOf, runs: paragraph.Runs}
}

// features returns the formatting marks that cover at least half of the
// visible characters in text[start:end].
func (styled styledText) features(start, end int) map[string]bool {
	if start < 0 || end > len(styled.text) || start >= end {
		return nil
	}

	total := 0
	counts := make(map[string]int)
	for index := start; index < end; {
		symbol, size := utf8.DecodeRuneInString(styled.text[index:end])
		if !unicode.IsSpace(symbol) {
			total++
			run := styled.runs[styled.runOf[index]]
			if run.Bold {
				counts["bold"]++
			}
			if run.Highlight != "" {
				counts["highlight:"+run.Highlight]++
			}
			if run.Color != "" {
				counts["color:"+run.Color]++
			}
			if run.Underline {
				counts["underline"]++
			}
			if run.Shading != "" {
				counts["shading:"+run.Shading]++
			}
		}
		index += size
	}

	if total == 0 {
		return nil
	}

	result := make(map[string]bool, len(counts))
	for feature, count := range counts {
		if count*2 >= total {
			result[feature] = true
		}
	}
	return result
}

type segment struct {
	value int
	start int
	end   int
}

type inlineLabel struct {
	value        int
	start        int
	contentStart int
}

func isLabelSeparator(symbol rune) bool {
	return unicode.IsSpace(symbol) || symbol == ';' || symbol == ',' ||
		symbol == ';'
}

func letterValue(symbol byte) int {
	switch {
	case symbol >= 'a' && symbol <= 'z':
		return int(symbol-'a') + 1
	case symbol >= 'A' && symbol <= 'Z':
		return int(symbol-'A') + 1
	}
	return 0
}

func skipSpaces(text string, index int) int {
	for index < len(text) {
		symbol, size := utf8.DecodeRuneInString(text[index:])
		if !unicode.IsSpace(symbol) {
			break
		}
		index += size
	}
	return index
}

// inlineLabels finds option labels such as "b)" or "C." inside text. A label
// must follow a separator (space, line break, ";" or ",").
func inlineLabels(text string, from int) []inlineLabel {
	labels := make([]inlineLabel, 0)
	for index := from; index < len(text); {
		symbol, size := utf8.DecodeRuneInString(text[index:])
		value := 0
		if symbol < utf8.RuneSelf {
			value = letterValue(byte(symbol))
		}
		if value >= 1 && value <= maxLetterValue {
			previousOK := index == 0
			if index > 0 {
				previous, _ := utf8.DecodeLastRuneInString(text[:index])
				previousOK = isLabelSeparator(previous)
			}
			if previousOK {
				next := index + 1
				for next < len(text) && (text[next] == ' ' || text[next] == '\t') {
					next++
				}
				end := -1
				if next < len(text) && text[next] == ')' {
					end = next + 1
				} else if index+1 < len(text) && text[index+1] == '.' {
					after := index + 2
					if after >= len(text) {
						end = after
					} else if following, _ := utf8.DecodeRuneInString(text[after:]); unicode.IsSpace(following) {
						end = after
					}
				}
				if end > 0 {
					labels = append(labels, inlineLabel{
						value:        value,
						start:        index,
						contentStart: skipSpaces(text, end),
					})
					index = end
					continue
				}
			}
		}
		index += size
	}
	return labels
}

// splitInline splits text[start:] at inline labels of letters that were not
// used yet. The order of the source is kept even when the labels are not
// alphabetical.
func splitInline(text string, start int, firstValue int) []segment {
	segments := make([]segment, 0, 5)
	currentValue, currentStart := firstValue, start
	used := map[int]bool{firstValue: true}

	for _, label := range inlineLabels(text, start) {
		if label.start <= currentStart || used[label.value] {
			continue
		}
		segments = append(segments, segment{currentValue, currentStart, label.start})
		currentValue, currentStart = label.value, label.contentStart
		used[label.value] = true
	}

	return append(segments, segment{currentValue, currentStart, len(text)})
}

// trimInlineSeparator removes the separator between two options on one
// line: "x  ;B)" or "x;B)". A separator attached to the text and followed
// by spaces or a line break is kept, because code options end with ";".
func trimInlineSeparator(raw string) string {
	trimmed := strings.TrimRightFunc(raw, unicode.IsSpace)
	last, size := utf8.DecodeLastRuneInString(trimmed)
	if last != ';' && last != ',' && last != '\u037e' {
		return trimmed
	}
	rest := trimmed[:len(trimmed)-size]
	previous, _ := utf8.DecodeLastRuneInString(rest)
	if len(trimmed) == len(raw) || unicode.IsSpace(previous) {
		return strings.TrimRightFunc(rest, unicode.IsSpace)
	}
	return trimmed
}

func cleanOptionText(text string) string {
	text = strings.TrimSpace(text)
	if matches := trailingDifficultyPattern.FindStringSubmatch(text); len(matches) == 2 {
		text = strings.TrimSpace(matches[1])
	}
	if matches := trailingMetadataPattern.FindStringSubmatch(text); len(matches) == 2 {
		text = strings.TrimSpace(matches[1])
	}
	return text
}

type parsedOption struct {
	value    int
	label    string
	text     string
	features map[string]bool
}

func buildOptions(styled styledText, segments []segment, kind labelKind) []parsedOption {
	options := make([]parsedOption, 0, len(segments))
	for index, current := range segments {
		raw := styled.text[current.start:current.end]
		if index+1 < len(segments) {
			raw = trimInlineSeparator(raw)
		}
		text := cleanOptionText(raw)

		var features map[string]bool
		if text != "" {
			if offset := strings.Index(styled.text[current.start:], text); offset >= 0 {
				begin := current.start + offset
				features = styled.features(begin, begin+len(text))
			}
		}

		label := strconv.Itoa(current.value)
		if kind == labelLetter {
			label = string(rune('A' + current.value - 1))
		}

		options = append(options, parsedOption{
			value:    current.value,
			label:    label,
			text:     text,
			features: features,
		})
	}
	return options
}

type item struct {
	index   int
	styled  styledText
	kind    labelKind
	value   int
	label   string
	offset  int
	auto    bool
	key     string
	carrier []segment
	inRun   bool
	isKey   bool
}

func classify(paragraph Paragraph, styled styledText) (labelKind, int, int, bool) {
	if paragraph.ListLabel != "" {
		switch paragraph.ListKind {
		case ListNumber:
			return labelNumber, paragraph.ListValue, 0, true
		case ListLetter:
			return labelLetter, paragraph.ListValue, 0, true
		}
	}

	text := styled.text
	if match := manualNumberPattern.FindStringSubmatchIndex(text); match != nil {
		end := match[1]
		if end >= len(text) || text[end] < '0' || text[end] > '9' {
			value, err := strconv.Atoi(text[match[2]:match[3]])
			if err == nil {
				return labelNumber, value, skipSpaces(text, end), false
			}
		}
	}

	if match := manualLetterPattern.FindStringSubmatchIndex(text); match != nil {
		end := match[1]
		value := letterValue(text[match[2]])
		dotStyle := text[end-1] == '.'
		// "A." must be followed by a space ("A. Text"), unless the
		// paragraph continues with the next option ("A.text ⏎B. text"). This
		// keeps "A.........is a building" and "A.S. Pushkin" as text.
		if !dotStyle || end >= len(text) || unicode.IsSpace(firstRune(text[end:])) ||
			len(splitInline(text, end, value)) > 1 {
			return labelLetter, value, skipSpaces(text, end), false
		}
	}

	return labelNone, 0, 0, false
}

func firstRune(text string) rune {
	symbol, _ := utf8.DecodeRuneInString(text)
	return symbol
}

func buildItems(paragraphs []Paragraph, skip map[int]bool) []*item {
	items := make([]*item, 0, len(paragraphs))
	for index, paragraph := range paragraphs {
		if skip[index] {
			continue
		}
		styled := newStyledText(paragraph)
		kind, value, offset, auto := classify(paragraph, styled)
		if strings.TrimSpace(styled.text) == "" && !(auto && kind != labelNone) {
			continue
		}

		current := &item{
			index:  index,
			styled: styled,
			kind:   kind,
			value:  value,
			offset: offset,
			auto:   auto,
			key:    "manual",
		}
		if auto {
			current.key = paragraph.ListKey
			current.label = paragraph.ListLabel
		}

		// "1. first option ⏎B) second ⏎C) third": Word turned the first
		// option label into a list number.
		if kind == labelNumber {
			segments := splitInline(styled.text, offset, 1)
			if len(segments) >= 3 && segments[1].value == 2 {
				hasA := false
				for _, label := range inlineLabels(styled.text, offset) {
					if label.start >= segments[1].start {
						break
					}
					if label.value == 1 && label.start > offset {
						hasA = true
					}
				}
				if !hasA {
					current.carrier = segments
				}
			}
		}

		items = append(items, current)
	}

	// Consecutive ascending numbers ("1. x", "2. y", "3. z") are statements
	// of a stem or numbered options, not questions.
	sameRun := func(first, second *item) bool {
		return first.kind == labelNumber && second.kind == labelNumber &&
			first.carrier == nil && second.carrier == nil &&
			first.key == second.key && second.value == first.value+1
	}
	for index, current := range items {
		if current.kind != labelNumber || current.carrier != nil {
			continue
		}
		if index > 0 && sameRun(items[index-1], current) ||
			index+1 < len(items) && sameRun(current, items[index+1]) {
			current.inRun = true
		}
	}

	return items
}

func (current *item) optionLike() bool {
	return current.kind == labelLetter || current.carrier != nil
}

func (current *item) content() string {
	return current.styled.text[current.offset:]
}

// stemLine renders a paragraph of the stem, keeping automatic numbers that
// the reader needs ("1. a white coat   a. mechanic").
func (current *item) stemLine() string {
	text := strings.TrimSpace(current.styled.text)
	if current.auto && current.label != "" {
		return strings.TrimSpace(current.label + " " + text)
	}
	return text
}

type block struct {
	question *item
	body     []*item
	heading  bool
}

func (b *block) hasOptions() bool {
	for _, current := range b.body {
		if current.optionLike() {
			return true
		}
	}
	return false
}

func buildBlocks(items []*item) ([]*block, []*item) {
	blocks := make([]*block, 0)
	preamble := make([]*item, 0)
	var current, lastReal *block

	for _, it := range items {
		candidate := it.kind == labelNumber && it.carrier == nil && !it.inRun && !it.isKey
		if candidate {
			switch {
			case current == nil:
			case current.hasOptions():
				lastReal = current
			case it.value == current.question.value+1 && it.key == current.question.key:
				// The previous question has no options.
			case lastReal == nil && it.value == 1,
				lastReal != nil && it.value == lastReal.question.value+1:
				// The previous numbered paragraph was a section heading.
				current.heading = true
			default:
				current.body = append(current.body, it)
				continue
			}
			current = &block{question: it}
			blocks = append(blocks, current)
			continue
		}

		if current == nil {
			preamble = append(preamble, it)
		} else {
			current.body = append(current.body, it)
		}
	}

	return blocks, preamble
}

type optionGroup struct {
	kind    labelKind
	options []parsedOption
	items   []*item
	next    int
	head    bool
}

type flatEntry struct {
	item    *item
	options []parsedOption
	value   int
	kind    labelKind
}

type rawQuestion struct {
	paragraph   int
	number      int
	head        string
	stem        []string
	options     []parsedOption
	trailing    []*item
	stemItems   []*item
	heading     bool
	skippedList int
}

func maxValue(options []parsedOption) int {
	result := 0
	for _, option := range options {
		result = max(result, option.value)
	}
	return result
}

func parseBlock(b *block) rawQuestion {
	question := b.question
	text := question.styled.text
	head := text[question.offset:]

	flat := make([]flatEntry, 0, len(b.body))
	for _, it := range b.body {
		switch {
		case it.isKey:
			continue
		case it.carrier != nil:
			flat = append(flat, flatEntry{it, buildOptions(it.styled, it.carrier, labelLetter), 1, labelLetter})
		case it.kind == labelLetter:
			segments := splitInline(it.styled.text, it.offset, it.value)
			flat = append(flat, flatEntry{it, buildOptions(it.styled, segments, labelLetter), it.value, labelLetter})
		case it.kind == labelNumber:
			segments := []segment{{it.value, it.offset, len(it.styled.text)}}
			flat = append(flat, flatEntry{it, buildOptions(it.styled, segments, labelNumber), it.value, labelNumber})
		default:
			flat = append(flat, flatEntry{item: it})
		}
	}

	groups := make([]*optionGroup, 0)
	var group *optionGroup

	// Options that start on the question line: "12. Question?  A) x  B) y".
	for _, label := range inlineLabels(text, question.offset) {
		if label.value != 1 || label.start <= question.offset {
			continue
		}
		segments := splitInline(text, label.contentStart, 1)
		continues := len(flat) > 0 && flat[0].options != nil &&
			flat[0].value == 2 && flat[0].kind == labelLetter
		if len(segments) > 1 || continues {
			head = text[question.offset:label.start]
			group = &optionGroup{
				kind:    labelLetter,
				options: buildOptions(question.styled, segments, labelLetter),
				head:    true,
			}
			group.next = maxValue(group.options) + 1
			groups = append(groups, group)
		}
		break
	}

	for index, entry := range flat {
		if entry.options == nil {
			group = nil
			groups = append(groups, &optionGroup{kind: labelNone, items: []*item{entry.item}})
			continue
		}

		startNew := group == nil || group.kind != entry.kind
		if !startNew && entry.value != group.next {
			restart := false
			if entry.value == 1 {
				if len(entry.options) > 1 {
					restart = true
				} else if index+1 < len(flat) {
					next := flat[index+1]
					restart = next.options != nil && next.value == 2 && next.kind == entry.kind
				}
			}
			startNew = restart
		}
		if startNew {
			group = &optionGroup{kind: entry.kind}
			groups = append(groups, group)
		}
		group.options = append(group.options, entry.options...)
		group.items = append(group.items, entry.item)
		group.next = max(group.next, maxValue(entry.options)+1)
	}

	chosen := -1
	for index, candidate := range groups {
		if candidate.kind == labelLetter && len(candidate.options) >= 2 {
			chosen = index
		}
	}
	if chosen < 0 {
		for index, candidate := range groups {
			if candidate.kind == labelNumber && len(candidate.options) >= 3 &&
				len(candidate.options) <= 10 && candidate.options[0].value == 1 {
				chosen = index
			}
		}
	}

	result := rawQuestion{
		paragraph: question.index,
		number:    question.value,
		heading:   b.heading,
	}

	if len(groups) > 0 && groups[0].head && chosen != 0 {
		head = text[question.offset:]
	}
	result.head = strings.TrimSpace(head)

	if chosen < 0 {
		for _, candidate := range groups {
			result.trailing = append(result.trailing, candidate.items...)
		}
		return result
	}

	for _, candidate := range groups[:chosen] {
		for _, it := range candidate.items {
			result.stem = append(result.stem, it.stemLine())
			result.stemItems = append(result.stemItems, it)
		}
	}
	result.options = groups[chosen].options
	for _, candidate := range groups[chosen+1:] {
		result.trailing = append(result.trailing, candidate.items...)
		if candidate.kind != labelNone {
			result.skippedList += len(candidate.items)
		}
	}

	return result
}

// stripQuestionMarkers removes difficulty markers that the document puts
// after most questions: "Question? (a)" or "Question ....  O".
func stripQuestionMarkers(questions []rawQuestion) {
	if len(questions) == 0 {
		return
	}

	parenCount, letterCount := 0, 0
	for _, question := range questions {
		if parenDifficultyPattern.MatchString(question.head) {
			parenCount++
		}
		if letterDifficultyPattern.MatchString(question.head) {
			letterCount++
		}
	}

	paren := parenCount*2 >= len(questions)
	letter := !paren && letterCount*2 >= len(questions)

	for index := range questions {
		head := questions[index].head
		switch {
		case paren:
			head = parenDifficultyPattern.ReplaceAllString(head, "")
		case letter:
			head = stripTrailingDifficultyLetter(head)
		}
		if matches := trailingMetadataPattern.FindStringSubmatch(head); len(matches) == 2 {
			head = matches[1]
		}
		questions[index].head = strings.TrimSpace(head)
	}
}

func stripTrailingDifficultyLetter(text string) string {
	trimmed := strings.TrimRightFunc(text, unicode.IsSpace)
	last, size := utf8.DecodeLastRuneInString(trimmed)
	if last != 'A' && last != 'O' && last != 'Ç' {
		return text
	}
	rest := trimmed[:len(trimmed)-size]
	if previous, _ := utf8.DecodeLastRuneInString(rest); rest != "" && unicode.IsUpper(previous) {
		return text
	}
	return strings.TrimRightFunc(rest, unicode.IsSpace)
}

func (question rawQuestion) text() string {
	lines := make([]string, 0, 1+len(question.stem))
	for _, line := range append([]string{question.head}, question.stem...) {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// segmentQuestions turns paragraphs into raw questions. Blocks without
// options that do not continue the question numbering are section headings.
func segmentQuestions(items []*item) ([]rawQuestion, []*item, []rawQuestion) {
	blocks, preamble := buildBlocks(items)

	parsed := make([]rawQuestion, 0, len(blocks))
	for _, b := range blocks {
		parsed = append(parsed, parseBlock(b))
	}

	questions := make([]rawQuestion, 0, len(parsed))
	skipped := make([]rawQuestion, 0)
	for index, question := range parsed {
		if len(question.options) == 0 {
			heading := question.heading
			if len(questions) > 0 && question.number != questions[len(questions)-1].number+1 {
				heading = true
			}
			if index+1 < len(parsed) {
				next := parsed[index+1].number
				if next != question.number+1 && next != 1 {
					heading = true
				}
			}
			if heading {
				skipped = append(skipped, question)
				continue
			}
		}
		questions = append(questions, question)
	}

	stripQuestionMarkers(questions)
	return questions, preamble, skipped
}

// ParseQuestions keeps the previous API: it parses with default options and
// drops the parser issues.
func ParseQuestions(
	paragraphs []Paragraph,
) []QuestionDraft {
	return Parse(paragraphs, ParseOptions{}).Questions
}
