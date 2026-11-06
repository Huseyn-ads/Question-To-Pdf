package importer

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Answer modes accepted by Preview.
const (
	AnswerModeAuto = "auto"
	AnswerModeAllA = "all_a"
)

var ErrInvalidAnswerMode = errors.New(
	`answer_mode must be "auto" or "all_a"`,
)

// NormalizeAnswerMode validates the answer_mode request value. An empty
// value means AnswerModeAuto.
func NormalizeAnswerMode(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", AnswerModeAuto:
		return AnswerModeAuto, nil
	case AnswerModeAllA:
		return AnswerModeAllA, nil
	}
	return "", ErrInvalidAnswerMode
}

type ParseOptions struct {
	// AnswerMode is AnswerModeAuto (author marks and explicit keys) or
	// AnswerModeAllA (option A is correct in every question).
	AnswerMode string
}

type ParseResult struct {
	Questions []QuestionDraft
	Issues    []ValidationIssue
}

// Issue codes produced by the parser in addition to ValidateQuestions.
const (
	IssueCorrectAnswerMissing    = "correct_answer_missing"
	IssueCorrectAnswerAmbiguous  = "correct_answer_ambiguous"
	IssueAnswerKeyInvalid        = "answer_key_invalid"
	IssueAnswerKeyConflict       = "answer_key_conflict"
	IssueAnswerModeLabelMissing  = "answer_mode_label_missing"
	IssueAnswerModeLabelRepeated = "answer_mode_label_duplicate"
	IssueEmptyOptionSkipped      = "empty_option_skipped"
	IssueContentSkipped          = "content_skipped"
	IssueSourceNumbersRenumbered = "source_numbers_renumbered"
)

var (
	// "Cavab: B", "Düzgün cavab - C", "Answer: (d)", "Ответ: A".
	questionKeyPattern = regexp.MustCompile(
		`(?i)^[\s\p{Zs}]*(?:(?:düzgün|doğru|correct|правильный)[\s\p{Zs}]+)?(?:cavab|answer|ответ|key|açar)[\s\p{Zs}]*[:\-–—][\s\p{Zs}]*\(?([A-Ha-h])\)?[\s\p{Zs}]*[.;]?[\s\p{Zs}]*$`,
	)

	keyHeadingPattern = regexp.MustCompile(
		`(?i)^[\s\p{Zs}]*(?:cavablar|düzgün[\s\p{Zs}]+cavablar|doğru[\s\p{Zs}]+cavablar|cavab(?:lar)?(?:ın)?[\s\p{Zs}]+açarı|açar|answer[\s\p{Zs}]+key|answers|key|ответы|ключ(?:[\s\p{Zs}]+ответов)?|правильные[\s\p{Zs}]+ответы)[\s\p{Zs}]*[:.]?[\s\p{Zs}]*$`,
	)

	keyTokenPattern = regexp.MustCompile(
		`(\d{1,4})[\s\p{Zs}]*[-–—.:)=][\s\p{Zs}]*([A-Ha-h])`,
	)

	keyTableTokenPattern = regexp.MustCompile(
		`(\d{1,4})[\s\p{Zs}]*[-–—.:)=]?[\s\p{Zs}]*([A-Ha-h])`,
	)

	correctWordPattern = regexp.MustCompile(
		`(?i)(?:düzgün|doğru)[\s\p{Zs}]+(?:cavab|variant)|правильн\p{L}*[\s\p{Zs}]+(?:ответ|вариант)|correct[\s\p{Zs}]+(?:answer|option)`,
	)

	everywherePattern = regexp.MustCompile(
		`(?i)hər[\s\p{Zs}]+yerdə|hər[\s\p{Zs}]+sual|bütün|hamısı|(?:^|[^\p{L}])(?:везде|всех|все|всегда)(?:[^\p{L}]|$)|\balways\b|\ball\b|\bevery\b`,
	)

	letterAPattern = regexp.MustCompile(
		`(?:^|[^\p{L}\p{N}])[AА](?:[\s\p{Zs}]*$|[\s\p{Zs}]*[.,;:!)»"”'’]|-|[\s\p{Zs}]+(?:variant|вариант|option))`,
	)
)

type answerKey struct {
	byNumber   map[int]string
	byPosition map[int]string
	allA       bool
	allASource string
}

// keyTokens returns number→letter pairs when text consists only of key
// entries ("1-A, 2-C; 3) B").
func keyTokens(text string, pattern *regexp.Regexp) map[int]string {
	matches := pattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	result := make(map[int]string, len(matches))
	previous := 0
	for _, match := range matches {
		if strings.TrimFunc(text[previous:match[0]], isKeySeparator) != "" {
			return nil
		}
		if next, _ := utf8.DecodeRuneInString(text[match[1]:]); match[1] < len(text) && unicode.IsLetter(next) {
			return nil
		}
		number, err := strconv.Atoi(text[match[2]:match[3]])
		if err != nil {
			return nil
		}
		result[number] = text[match[4]:match[5]]
		previous = match[1]
	}
	if strings.TrimFunc(text[previous:], isKeySeparator) != "" {
		return nil
	}

	return result
}

// keyTokens keeps the letters as written; a key without a heading must use
// capital letters, which separates "1-A 2-C 3-B" from matching options such
// as "1-a, 2-c, 3-b".
func allUpper(tokens map[int]string) bool {
	return tokens != nil && !strings.ContainsFunc(joinValues(tokens), unicode.IsLower)
}

func joinValues(tokens map[int]string) string {
	var builder strings.Builder
	for _, value := range tokens {
		builder.WriteString(value)
	}
	return builder.String()
}

func isKeySeparator(symbol rune) bool {
	return unicode.IsSpace(symbol) || strings.ContainsRune(",;.|/", symbol)
}

// findAnswerKey detects a separate answer key: a heading such as "Cavablar"
// or "Answer key" followed by entries, or standalone lines with at least
// three "N-L" entries. Table rows are read as one line. The returned set
// marks paragraphs that belong to the key.
func findAnswerKey(paragraphs []Paragraph) (map[int]string, map[int]bool) {
	type line struct {
		indexes []int
		text    string
		table   bool
		// listValue is the automatic number of a list paragraph.
		listValue int
		labelled  bool
	}

	lines := make([]line, 0, len(paragraphs))
	for index := 0; index < len(paragraphs); index++ {
		paragraph := paragraphs[index]
		row := paragraph.TableRow
		if row == 0 {
			current := line{indexes: []int{index}, text: paragraph.Text()}
			if paragraph.ListLabel != "" && paragraph.ListKind != ListNone {
				current.labelled = true
				current.listValue = paragraph.ListValue
			}
			if kind, _, _, _ := classify(Paragraph{Runs: paragraph.Runs}, newStyledText(paragraph)); kind == labelLetter {
				current.labelled = true
			}
			lines = append(lines, current)
			continue
		}
		current := line{table: true}
		cells := make([]string, 0, 2)
		for index < len(paragraphs) && paragraphs[index].TableRow == row {
			current.indexes = append(current.indexes, index)
			cells = append(cells, strings.TrimSpace(paragraphs[index].Text()))
			index++
		}
		index--
		current.text = strings.Join(cells, "\t")
		lines = append(lines, current)
	}

	key := make(map[int]string)
	members := make(map[int]bool)
	inSection := false

	for _, current := range lines {
		text := strings.TrimSpace(current.text)
		if text == "" {
			continue
		}
		if keyHeadingPattern.MatchString(text) {
			inSection = true
			for _, index := range current.indexes {
				members[index] = true
			}
			continue
		}

		pattern := keyTokenPattern
		if current.table {
			pattern = keyTableTokenPattern
		}
		var tokens map[int]string
		switch {
		case current.labelled && inSection && current.listValue > 0:
			// A numbered list inside the key section: "1. B".
			if letter := strings.Trim(text, " \t.;,)"); len(letter) == 1 && letterValue(letter[0]) >= 1 &&
				letterValue(letter[0]) <= maxLetterValue {
				tokens = map[int]string{current.listValue: strings.ToUpper(letter)}
			}
		case current.labelled:
			// Option paragraphs such as "a) 1-a, 2-b, 3-c" are not keys.
		default:
			tokens = keyTokens(text, pattern)
			if !inSection && !current.table && !allUpper(tokens) {
				tokens = nil
			}
		}
		if tokens == nil && current.table && inSection {
			// Header row of a key table, for example "Sual | Cavab".
			if !strings.ContainsFunc(text, unicode.IsDigit) {
				for _, index := range current.indexes {
					members[index] = true
				}
				continue
			}
		}
		if tokens == nil || (!inSection && len(tokens) < 3) {
			inSection = false
			continue
		}

		for number, letter := range tokens {
			key[number] = strings.ToUpper(letter)
		}
		for _, index := range current.indexes {
			members[index] = true
		}
	}

	return key, members
}

func isAllAStatement(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > 200 {
		return false
	}
	return correctWordPattern.MatchString(text) &&
		everywherePattern.MatchString(text) &&
		letterAPattern.MatchString(text)
}

type finalQuestion struct {
	raw     rawQuestion
	number  int
	key     string
	options []parsedOption
	skipped []string
}

// Parse extracts questions from paragraphs and decides which options are
// correct. Correct answers come from, in this order: an explicit key line in
// the question, a separate answer key, a document note saying that A is
// always correct, and finally the author's formatting mark (bold,
// highlight, colour...). With AnswerModeAllA the option labelled A is
// correct and every disagreement is reported.
func Parse(paragraphs []Paragraph, options ParseOptions) ParseResult {
	mode, err := NormalizeAnswerMode(options.AnswerMode)
	if err != nil {
		mode = AnswerModeAuto
	}

	issues := make([]ValidationIssue, 0)

	documentKey, keyParagraphs := findAnswerKey(paragraphs)
	items := buildItems(paragraphs, keyParagraphs)

	// Per-question key lines are plain paragraphs, never options.
	questionKeys := make(map[*item]string)
	for _, it := range items {
		if it.kind == labelNone || it.kind == labelLetter && !it.auto {
			if match := questionKeyPattern.FindStringSubmatch(it.styled.text); match != nil {
				it.isKey = true
				questionKeys[it] = strings.ToUpper(match[1])
			}
		}
	}

	raws, preamble, skipped := segmentQuestions(items)

	allA := false
	for _, it := range preamble {
		if isAllAStatement(it.styled.text) {
			allA = true
		}
	}
	for _, question := range raws {
		for _, it := range question.trailing {
			if it.kind == labelNone && isAllAStatement(it.styled.text) {
				allA = true
			}
		}
	}
	for index := range keyParagraphs {
		if isAllAStatement(paragraphs[index].Text()) {
			allA = true
		}
	}

	for _, question := range skipped {
		message := fmt.Sprintf(
			"numbered paragraph %q has no answer options and was not imported",
			shorten(strconv.Itoa(question.number)+". "+question.text(), 80),
		)
		if len(question.trailing) > 0 {
			message += fmt.Sprintf(" (with %d following paragraphs)", len(question.trailing))
		}
		issues = append(issues, ValidationIssue{
			Code:    IssueContentSkipped,
			Message: message,
		})
	}

	// Source numbers: the printed numbers when they are unique, otherwise the
	// position of the question in the document.
	printed := make(map[int]bool, len(raws))
	unique := true
	for _, question := range raws {
		if printed[question.number] || question.number < 1 {
			unique = false
		}
		printed[question.number] = true
	}
	if !unique && len(raws) > 0 {
		issues = append(issues, ValidationIssue{
			Field: "source_number",
			Code:  IssueSourceNumbersRenumbered,
			Message: "question numbering restarts or repeats in the document; " +
				"source_number is the position of the question in the document",
		})
	}

	marker := chooseMarker(raws)

	drafts := make([]QuestionDraft, 0, len(raws))
	answerIssues := make(map[int]bool)

	for position, question := range raws {
		number := question.number
		if !unique {
			number = position + 1
		}

		// Key lines were excluded from the stem and the options by
		// parseBlock; find them in the paragraph range of this question.
		keyLetter := questionKeyBetween(items, questionKeys, question.paragraph, nextParagraph(raws, position))
		if keyLetter == "" && len(documentKey) > 0 {
			if unique {
				keyLetter = documentKey[question.number]
			} else {
				keyLetter = documentKey[position+1]
			}
		}
		keySource := "the answer key"
		if keyLetter == "" && allA {
			keyLetter = "A"
			keySource = "the note that A is always correct"
		}

		kept := make([]parsedOption, 0, len(question.options))
		for _, option := range question.options {
			if option.text == "" {
				issues = append(issues, ValidationIssue{
					SourceNumber: number,
					Field:        "options",
					Code:         IssueEmptyOptionSkipped,
					Message:      fmt.Sprintf("empty option %s was skipped", option.label),
				})
				continue
			}
			kept = append(kept, option)
		}

		if question.skippedList > 0 {
			issues = append(issues, ValidationIssue{
				SourceNumber: number,
				Code:         IssueContentSkipped,
				Message: fmt.Sprintf(
					"%d numbered paragraphs after the options were not imported",
					question.skippedList,
				),
			})
		}

		marked := make([]bool, len(kept))
		markedCount := 0
		for index, option := range kept {
			if marker != "" && option.features[marker] {
				marked[index] = true
				markedCount++
			}
		}
		if markedCount == len(kept) {
			// The whole block is formatted the same way: no mark.
			for index := range marked {
				marked[index] = false
			}
			markedCount = 0
		}

		correct := make([]bool, len(kept))
		addIssue := func(code, message string) {
			issues = append(issues, ValidationIssue{
				SourceNumber: number,
				Field:        "options",
				Code:         code,
				Message:      message,
			})
			answerIssues[number] = true
		}
		labelIndexes := func(label string) []int {
			result := make([]int, 0, 1)
			for index, option := range kept {
				if option.label == label {
					result = append(result, index)
				}
			}
			return result
		}
		markedLabels := func() string {
			labels := make([]string, 0, markedCount)
			for index, isMarked := range marked {
				if isMarked {
					labels = append(labels, kept[index].label)
				}
			}
			return strings.Join(labels, ", ")
		}

		switch {
		case mode == AnswerModeAllA:
			indexes := labelIndexes("A")
			switch len(indexes) {
			case 0:
				addIssue(IssueAnswerModeLabelMissing,
					"answer_mode all_a: the question has no option labelled A; no option is marked correct")
			case 1:
				correct[indexes[0]] = true
			default:
				addIssue(IssueAnswerModeLabelRepeated,
					fmt.Sprintf("answer_mode all_a: option label A appears %d times; no option is marked correct", len(indexes)))
			}
			if keyLetter != "" && keyLetter != "A" {
				addIssue(IssueAnswerKeyConflict,
					fmt.Sprintf("answer_mode all_a conflicts with %s, which gives %s", keySource, keyLetter))
			}
			if markedCount > 0 && (len(indexes) != 1 || markedCount != 1 || !marked[indexes[0]]) {
				addIssue(IssueAnswerKeyConflict,
					fmt.Sprintf("answer_mode all_a conflicts with the author's mark on option %s", markedLabels()))
			}

		case keyLetter != "":
			indexes := labelIndexes(keyLetter)
			switch len(indexes) {
			case 0:
				addIssue(IssueAnswerKeyInvalid,
					fmt.Sprintf("%s gives %s, but the question has no option %s; no option is marked correct", keySource, keyLetter, keyLetter))
			case 1:
				correct[indexes[0]] = true
				if markedCount > 0 && (markedCount != 1 || !marked[indexes[0]]) {
					addIssue(IssueAnswerKeyConflict,
						fmt.Sprintf("%s gives %s, but the author marked option %s; the key is used", keySource, keyLetter, markedLabels()))
				}
			default:
				addIssue(IssueCorrectAnswerAmbiguous,
					fmt.Sprintf("%s gives %s, but option label %s appears %d times; no option is marked correct", keySource, keyLetter, keyLetter, len(indexes)))
			}

		case markedCount == 0:
			addIssue(IssueCorrectAnswerMissing,
				"no correct option is marked in the document; options are left unmarked")

		default:
			copy(correct, marked)
			if markedCount > 1 {
				addIssue(IssueCorrectAnswerAmbiguous,
					fmt.Sprintf("the author marked %d options (%s) as correct", markedCount, markedLabels()))
			}
		}

		draft := QuestionDraft{
			SourceNumber: number,
			Text:         question.text(),
			Options:      make([]OptionDraft, 0, len(kept)),
		}
		for index, option := range kept {
			draft.Options = append(draft.Options, OptionDraft{
				Label:     option.label,
				Text:      option.text,
				IsCorrect: correct[index],
			})
		}
		drafts = append(drafts, draft)
	}

	validation := ValidateQuestions(drafts)
	for _, issue := range validation {
		if issue.Code == "exactly_one_correct_option" && answerIssues[issue.SourceNumber] {
			continue
		}
		issues = append(issues, issue)
	}

	sort.SliceStable(issues, func(left, right int) bool {
		return issues[left].SourceNumber < issues[right].SourceNumber
	})

	return ParseResult{Questions: drafts, Issues: issues}
}

func nextParagraph(raws []rawQuestion, position int) int {
	if position+1 < len(raws) {
		return raws[position+1].paragraph
	}
	return int(^uint(0) >> 1)
}

func questionKeyBetween(items []*item, keys map[*item]string, from, to int) string {
	result := ""
	for _, it := range items {
		if it.index > from && it.index < to {
			if letter, ok := keys[it]; ok {
				result = letter
			}
		}
	}
	return result
}

// chooseMarker picks the formatting that marks exactly one option in the
// largest number of questions.
func chooseMarker(questions []rawQuestion) string {
	counts := make(map[string]int)
	for _, question := range questions {
		seen := make(map[string]int)
		for _, option := range question.options {
			for feature := range option.features {
				seen[feature]++
			}
		}
		for feature, count := range seen {
			if count == 1 {
				counts[feature]++
			}
		}
	}

	priority := func(feature string) int {
		switch {
		case feature == "bold":
			return 0
		case strings.HasPrefix(feature, "highlight:"):
			return 1
		case strings.HasPrefix(feature, "color:"):
			return 2
		case feature == "underline":
			return 3
		}
		return 4
	}

	features := make([]string, 0, len(counts))
	for feature := range counts {
		features = append(features, feature)
	}
	slices.SortFunc(features, func(left, right string) int {
		if counts[left] != counts[right] {
			return counts[right] - counts[left]
		}
		if priority(left) != priority(right) {
			return priority(left) - priority(right)
		}
		return strings.Compare(left, right)
	})

	if len(features) == 0 {
		return ""
	}
	return features[0]
}

func shorten(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit]) + "…"
}

// BuildPreview parses paragraphs into the preview response.
func BuildPreview(paragraphs []Paragraph, options ParseOptions) PreviewResult {
	result := Parse(paragraphs, options)
	return PreviewResult{
		QuestionsCount: len(result.Questions),
		CanImport:      true,
		Issues:         result.Issues,
		Questions:      result.Questions,
	}
}
