package importer

import (
	"strings"
	"testing"
)

// Each test reproduces one layout found in the exam documents.

func TestManualNumbersBoldMarkAndDifficulty(t *testing.T) {
	// "FƏLSƏFƏ TEST 300": numbers typed as text, the correct option is bold,
	// a difficulty letter follows some options after many spaces.
	body := para(run("1. Fəlsəfənin predmetinə nə daxildir?   ")) +
		para(run("\t\tA) Yalnız təbiət elmləri")) +
		para(run("\t\tB) İnsan təcrübəsi və varlıq məsələləri", propBold), run("                    A", propBold)) +
		para(run("\t\tC) Rəssamlıq və musiqi                    O")) +
		para(run("2.\tİkinci sual?")) +
		para(run("A) Bir"), run("        "), run("B) İki", propBold)) +
		para(run("C) Üç"))

	result := parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{1, "Fəlsəfənin predmetinə nə daxildir?", []string{"A:Yalnız təbiət elmləri", "*B:İnsan təcrübəsi və varlıq məsələləri", "C:Rəssamlıq və musiqi"}},
		{2, "İkinci sual?", []string{"A:Bir", "*B:İki", "C:Üç"}},
	})
	checkIssues(t, result.Issues, map[string][]int{})
}

func TestWordListsHighlightMarkAndParenDifficulty(t *testing.T) {
	// "Alq və prq 325": question and option numbers exist only in
	// numbering.xml, the correct option is highlighted (not bold) and the
	// bold question ends with a difficulty marker "(a)".
	numbering := numberingXML(
		list{1, []listLevel{decimalDot}},
		list{2, []listLevel{upperDot}},
		list{3, []listLevel{upperDot}},
	)
	body := listPara(1, 0, run("Alqoritm termini hansı alimin adı ilə bağlıdır?", propBold), run(" (a)", propBold)) +
		listPara(2, 0, run("Nəsirəddin Tusi")) +
		listPara(2, 0, run("Əl-Xarəzmi", propHighlight)) +
		listPara(2, 0, run("İbn Sina")) +
		listPara(1, 0, run("Sonluluq xassəsi nədir?", propBold), run("(ç)", propBold)) +
		listPara(3, 0, run("Dayanmır")) +
		listPara(3, 0, run("Sonlu sayda addımdan sonra dayanır", propHighlight)) +
		para(run("Top of Form", propHidden))

	result := parseDocx(t, body, numbering, "")
	checkQuestions(t, result.Questions, []want{
		{1, "Alqoritm termini hansı alimin adı ilə bağlıdır?", []string{"A:Nəsirəddin Tusi", "*B:Əl-Xarəzmi", "C:İbn Sina"}},
		{2, "Sonluluq xassəsi nədir?", []string{"A:Dayanmır", "*B:Sonlu sayda addımdan sonra dayanır"}},
	})
	checkIssues(t, result.Issues, map[string][]int{})
}

func TestRedColourMarkAndLetterDifficulty(t *testing.T) {
	// "Yumşaq bacarıqlar": red font marks the answer (different red
	// shades), questions end with a difficulty letter A/O/Ç.
	numbering := numberingXML(
		list{144, []listLevel{decimalParen}},
		list{1, []listLevel{lowerParen}},
		list{2, []listLevel{lowerParen}},
	)
	body := listPara(144, 0, run("Yumşaq bacarıqlar dəstinə daxil olan xüsusiyyət ....... \tA", propBold)) +
		listPara(1, 0, run("fərqindəlikdir ", propRed)) +
		listPara(1, 0, run("belə xüsusiyyət yoxdur")) +
		listPara(144, 0, run("Tənqidi düşünən insan ........O", propBold)) +
		listPara(2, 0, run("hər məlumatı qəbul edir")) +
		listPara(2, 0, run("faktları yoxlayır", propDarkRed))

	result := parseDocx(t, body, numbering, "")
	checkQuestions(t, result.Questions, []want{
		{1, "Yumşaq bacarıqlar dəstinə daxil olan xüsusiyyət .......", []string{"*A:fərqindəlikdir", "B:belə xüsusiyyət yoxdur"}},
		{2, "Tənqidi düşünən insan ........", []string{"A:hər məlumatı qəbul edir", "*B:faktları yoxlayır"}},
	})
}

func TestInlineOptionsWithListLabelAndMatchingStem(t *testing.T) {
	// "+IAK": a sentence stem, then options in one paragraph whose first
	// label "a)" is a Word list number; matching questions keep the
	// numbered statements in the question text; inline labels may be out
	// of alphabetical order.
	numbering := numberingXML(
		list{1, []listLevel{decimalParen}},
		list{3, []listLevel{lowerParen}},
		list{25, []listLevel{decimalDot}},
		list{26, []listLevel{lowerParen}},
		list{32, []listLevel{lowerDot}},
		list{33, []listLevel{lowerParen}},
		list{40, []listLevel{lowerParen}},
	)
	body := listPara(1, 0, run("Complete the sentence")) +
		para(run("If we are polite, people will have ______ in us.")) +
		listPara(3, 0, run("attitude    b) torn     "), run("c)", propBold), run(" "), run("confidence     ", propBold), run("d) avoid    e) expect")) +
		listPara(1, 0, run("Answer the question with the given words")) +
		para(run("    Who wears?")) +
		listPara(25, 0, run("a white coat           a. mechanic")) +
		listPara(25, 0, run("overalls                  b. a physician")) +
		listPara(26, 0, run("1b, 2a", propBold)) +
		listPara(26, 0, run("1a, 2b")) +
		listPara(1, 0, run("Match the adjectives")) +
		listPara(32, 0, run("bad mannered     1. mean")) +
		listPara(32, 0, run("earnest          2. well mannered")) +
		listPara(33, 0, run("1a, 2b")) +
		listPara(33, 0, run("1b, 2a", propBold)) +
		listPara(1, 0, run("Choose the correct variant")) +
		listPara(40, 0, run("evacuate", propBold), run("   c) flood    b) warning"))

	result := parseDocx(t, body, numbering, "")
	checkQuestions(t, result.Questions, []want{
		{1, "Complete the sentence\nIf we are polite, people will have ______ in us.",
			[]string{"A:attitude", "B:torn", "*C:confidence", "D:avoid", "E:expect"}},
		{2, "Answer the question with the given words\nWho wears?\n1. a white coat           a. mechanic\n2. overalls                  b. a physician",
			[]string{"*A:1b, 2a", "B:1a, 2b"}},
		{3, "Match the adjectives\na. bad mannered     1. mean\nb. earnest          2. well mannered",
			[]string{"A:1a, 2b", "*B:1b, 2a"}},
		{4, "Choose the correct variant", []string{"*A:evacuate", "C:flood", "B:warning"}},
	})
	checkIssues(t, result.Issues, map[string][]int{})
}

func TestNumericOptionsAndOptionsSplitByLineBreaks(t *testing.T) {
	// "+IAK": options numbered 1)–5). "KOLLOKVİUM": all options in one
	// paragraph separated by line breaks, the first option label turned
	// into a Word list number "1.".
	numbering := numberingXML(
		list{1, []listLevel{decimalDot, decimalDot}},
		list{46, []listLevel{decimalParen}},
	)
	body := listPara(1, 0, run("Put the words in the correct order")) +
		para(run("ambitious/am/I/an/not")) +
		listPara(46, 0, run("I am not ambitious an")) +
		listPara(46, 0, run("I am not an ambitious", propBold)) +
		listPara(46, 0, run("I am an ambitious not")) +
		listPara(1, 0, run("Proseslər bir-birini necə aradan götürür?", propBold), run("    (ç)", propBold)) +
		listPara(1, 1, run("Konflikt yaranır", propBold), run("\nB) Yaddaşı paylaşır\nC) Diskdən oxuyur")) +
		para(run("D)   Proses silinir")) +
		listPara(1, 0, run("Windows 10-da hansı funksiya pəncərəni yanına yerləşdirir?   (o)", propBold)) +
		para(run("A.Snap", propBold), run("\nB. Dock\nc. Align"))

	result := parseDocx(t, body, numbering, "")
	checkQuestions(t, result.Questions, []want{
		{1, "Put the words in the correct order\nambitious/am/I/an/not",
			[]string{"1:I am not ambitious an", "*2:I am not an ambitious", "3:I am an ambitious not"}},
		{2, "Proseslər bir-birini necə aradan götürür?",
			[]string{"*A:Konflikt yaranır", "B:Yaddaşı paylaşır", "C:Diskdən oxuyur", "D:Proses silinir"}},
		{3, "Windows 10-da hansı funksiya pəncərəni yanına yerləşdirir?",
			[]string{"*A:Snap", "B:Dock", "C:Align"}},
	})

	// "Alq və prq 325" Q251: code options keep their final ";".
	body = para(run("251. Struktur tipi necə elan edilir?")) +
		para(run("A.", propHighlight), run("struct t { int a, b; };", propHighlight),
			run("\nB. struct t { a, b; };\nc. struct t (a, b);"))
	result = parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{251, "Struktur tipi necə elan edilir?",
			[]string{"*A:struct t { int a, b; };", "B:struct t { a, b; };", "C:struct t (a, b);"}},
	})
}

func TestOptionsOnQuestionLineAndGreekQuestionMark(t *testing.T) {
	// "FƏLSƏFƏ TEST 300": options start on the question line and are
	// separated by ";" or the Greek question mark U+037E.
	body := para(run("72. Demokritin fəlsəfəsinin əsası nədir?     A) Məntiq     B) Ruh    "), run("C) Atomlar", propBold)) +
		para(run("              D) Tanrılar        E) İdeyalar")) +
		para(run("267) Şüurun qabiliyyəti:   A)Ağıldır    ;B) Dərrakədir;")) +
		para(run("C) Məntiqdir    "), run("D)İntuisiyadır;", propBold), run("   E)İntellektdir"))

	result := parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{72, "Demokritin fəlsəfəsinin əsası nədir?", []string{"A:Məntiq", "B:Ruh", "*C:Atomlar", "D:Tanrılar", "E:İdeyalar"}},
		{267, "Şüurun qabiliyyəti:", []string{"A:Ağıldır", "B:Dərrakədir;", "C:Məntiqdir", "*D:İntuisiyadır;", "E:İntellektdir"}},
	})
}

func TestSectionHeadingsAndRestartedNumbering(t *testing.T) {
	// Numbered section headings without options are not questions; a
	// numbered statement inside a stem is not a question; numbering that
	// restarts per section is replaced by the document position.
	body := para(run("1. FƏLSƏFƏ, ONUN PREDMETİ", propBold)) +
		para(run("1. Birinci sual?")) +
		para(run("A) Bir")) +
		para(run("B) İki", propBold)) +
		para(run("2. QƏDİM ŞƏRQ FƏLSƏFƏSİ")) +
		para(run("2. Samsung özünü belə tanıtmışdır")) +
		para(run("1. __________ : Yeni gələcək yaradan texnologiyalar")) +
		para(run("A) 1.vizyon")) +
		para(run("B) 1.missiya", propBold)) +
		para(run("\t\t\t(Çəki3)")) +
		para(run("Mövzu-2", propBold)) +
		para(run("1.Karyera yol xəritəniz hansıdır?")) +
		para(run("A) Planlama")) +
		para(run("B) Problemlər", propBold), run("\t\t\t(çətin tipli)"))

	result := parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{1, "Birinci sual?", []string{"A:Bir", "*B:İki"}},
		{2, "Samsung özünü belə tanıtmışdır\n1. __________ : Yeni gələcək yaradan texnologiyalar", []string{"A:1.vizyon", "*B:1.missiya"}},
		{3, "Karyera yol xəritəniz hansıdır?", []string{"A:Planlama", "*B:Problemlər"}},
	})
	checkIssues(t, result.Issues, map[string][]int{
		IssueContentSkipped:          {0, 0},
		IssueSourceNumbersRenumbered: {0},
	})
}

func TestEmptyAndStrayOptionsAreReported(t *testing.T) {
	// "Kompüter arxitekturası" Q134: an empty numbered option; "grafic"
	// Q141: a one-item list "A." after option E.
	numbering := numberingXML(
		list{145, []listLevel{decimalDot}},
		list{132, []listLevel{upperDot}},
		list{179, []listLevel{upperDot}},
	)
	body := listPara(145, 0, run("Hansı mouse növü top istifadə edir?(a)")) +
		listPara(132, 0, run("Optik Mouse")) +
		listPara(132, 0) +
		listPara(132, 0, run("Lazer Mouse")) +
		listPara(132, 0, run("Mexaniki Mouse", propBold)) +
		listPara(179, 0, run("30"))

	result := parseDocx(t, body, numbering, "")
	checkQuestions(t, result.Questions, []want{
		{1, "Hansı mouse növü top istifadə edir?", []string{"A:Optik Mouse", "C:Lazer Mouse", "*D:Mexaniki Mouse", "A:30"}},
	})
	checkIssues(t, result.Issues, map[string][]int{
		IssueEmptyOptionSkipped:  {1},
		"duplicate_option_label": {1},
	})
}

func TestCharacterStyleBoldAndExplicitNotBold(t *testing.T) {
	// "KOLLOKVİUM": bold comes from the character style "Strong"; a run with
	// the same style and <w:b w:val="0"/> is not bold.
	body := para(run("32. MS-DOS-da proqramların icrası üçün əmr hansıdır?", propStrong)) +
		para(run("A) "), run("COMMAND", propStrong)) +
		para(run("B) RUN")) +
		para(run("C) "), run("OPEN", propStrong, propNotBold))

	result := parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{32, "MS-DOS-da proqramların icrası üçün əmr hansıdır?", []string{"*A:COMMAND", "B:RUN", "C:OPEN"}},
	})
}

func TestStructuredTextAfterLastQuestionIsReported(t *testing.T) {
	// "Alq və prq 325": a list of open exam questions follows the tests.
	numbering := numberingXML(
		list{1, []listLevel{decimalDot}},
		list{2, []listLevel{upperDot}},
		list{9, []listLevel{decimalDot}},
	)
	body := listPara(1, 0, run("Massivə necə müraciət edilir?")) +
		listPara(2, 0, run("[]", propHighlight)) +
		listPara(2, 0, run("{}")) +
		para(run("İmtahan sualları", propBold)) +
		listPara(9, 0, run("Alqoritm anlayışı.")) +
		listPara(9, 0, run("Xətti alqoritmik struktur.")) +
		listPara(9, 0, run("Dövri alqoritmlər."))

	result := parseDocx(t, body, numbering, "")
	checkQuestions(t, result.Questions, []want{
		{1, "Massivə necə müraciət edilir?", []string{"*A:[]", "B:{}"}},
	})
	checkIssues(t, result.Issues, map[string][]int{IssueContentSkipped: {1}})
}

func TestRealDocumentInTestdata(t *testing.T) {
	paragraphs, err := ExtractParagraphs("testdata/FƏLSƏFƏ TEST 300.docx")
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(paragraphs, ParseOptions{})
	if len(result.Questions) != 300 {
		t.Fatalf("questions: got %d, want 300", len(result.Questions))
	}
	for index, question := range result.Questions {
		if question.SourceNumber != index+1 {
			t.Fatalf("question %d has source_number %d", index+1, question.SourceNumber)
		}
		correct := 0
		for _, option := range question.Options {
			if option.IsCorrect {
				correct++
			}
		}
		if len(question.Options) != 5 || correct != 1 {
			t.Fatalf("question %d: %d options, %d correct", question.SourceNumber, len(question.Options), correct)
		}
	}
	if got := result.Questions[71]; got.Text != "Qədim Yunan filosofu Demokritin fəlsəfəsinin əsasını nə təşkil edir?" ||
		got.Options[2].Text != "Atomlar" || !got.Options[2].IsCorrect {
		t.Fatalf("question 72: %+v", got)
	}
	codes := issueCodes(result.Issues)
	if len(codes[IssueContentSkipped]) != 14 || len(codes) != 2 || len(codes["duplicate_option_text"]) != 1 {
		t.Fatalf("issues: %v", codes)
	}
}

func TestNormalizeAnswerMode(t *testing.T) {
	for input, expected := range map[string]string{"": AnswerModeAuto, "auto": AnswerModeAuto, " ALL_A ": AnswerModeAllA} {
		got, err := NormalizeAnswerMode(input)
		if err != nil || got != expected {
			t.Fatalf("%q: %q %v", input, got, err)
		}
	}
	if _, err := NormalizeAnswerMode("all_b"); err == nil {
		t.Fatal("all_b accepted")
	}
}

func TestParseQuestionsKeepsOldAPI(t *testing.T) {
	paragraphs := []Paragraph{
		{Runs: []TextRun{{Text: "1. Paytaxt?"}}},
		{Runs: []TextRun{{Text: "A) Bakı", Bold: true}}},
		{Runs: []TextRun{{Text: "B) Gəncə"}}},
	}
	questions := ParseQuestions(paragraphs)
	if len(questions) != 1 || !questions[0].Options[0].IsCorrect || questions[0].Options[1].IsCorrect ||
		!strings.HasPrefix(questions[0].Text, "Paytaxt") {
		t.Fatalf("%+v", questions)
	}
}
