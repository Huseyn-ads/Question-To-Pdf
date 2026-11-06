package importer

import "testing"

// Sources of the correct answer: author marks, a separate key, a note that
// A is always correct, and the answer_mode=all_a preview option.

func unmarkedQuestions() string {
	return para(run("1. Azərbaycanın paytaxtı?")) +
		para(run("A) Bakı")) +
		para(run("B) Gəncə")) +
		para(run("C) Şəki")) +
		para(run("2. Ən uzun çay?")) +
		para(run("A) Kür")) +
		para(run("B) Araz")) +
		para(run("3. Ən hündür dağ?")) +
		para(run("B) Tufandağ")) +
		para(run("C) Bazardüzü")) +
		para(run("4. Rəngi seçin")) +
		para(run("A) Qırmızı")) +
		para(run("A) Yaşıl"))
}

func TestBoldMarksTheCorrectOption(t *testing.T) {
	body := para(run("1. Azərbaycanın paytaxtı?")) +
		para(run("A) "), run("Bakı", propBold)) +
		para(run("B) Gəncə")) +
		para(run("2. Ən uzun çay?")) +
		para(run("A) Kür", propBold)) +
		para(run("B) Araz", propBold)) +
		para(run("3. Ən hündür dağ?")) +
		para(run("A) Tufandağ")) +
		para(run("B) Bazardüzü"))

	result := parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{1, "Azərbaycanın paytaxtı?", []string{"*A:Bakı", "B:Gəncə"}},
		// Every option is bold: the formatting does not mark an answer.
		{2, "Ən uzun çay?", []string{"A:Kür", "B:Araz"}},
		{3, "Ən hündür dağ?", []string{"A:Tufandağ", "B:Bazardüzü"}},
	})
	checkIssues(t, result.Issues, map[string][]int{
		IssueCorrectAnswerMissing: {2, 3},
	})
}

func TestSeveralMarkedOptionsAreAmbiguous(t *testing.T) {
	body := para(run("1. Sual?")) +
		para(run("A) Bir", propHighlight)) +
		para(run("B) İki", propHighlight)) +
		para(run("C) Üç")) +
		para(run("2. Sual?")) +
		para(run("A) Bir")) +
		para(run("B) İki", propHighlight)) +
		para(run("C) Üç"))

	result := parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{1, "Sual?", []string{"*A:Bir", "*B:İki", "C:Üç"}},
		{2, "Sual?", []string{"A:Bir", "*B:İki", "C:Üç"}},
	})
	checkIssues(t, result.Issues, map[string][]int{IssueCorrectAnswerAmbiguous: {1}})
}

func TestSeparateAnswerKey(t *testing.T) {
	body := para(run("1. Azərbaycanın paytaxtı?")) +
		para(run("A) Bakı")) +
		para(run("B) Gəncə")) +
		para(run("2. Ən uzun çay?")) +
		para(run("A) Kür")) +
		para(run("B) Araz", propBold)) +
		para(run("3. Ən hündür dağ?")) +
		para(run("A) Tufandağ")) +
		para(run("B) Bazardüzü")) +
		para(run("4. Rəng?")) +
		para(run("A) Qırmızı")) +
		para(run("B) Yaşıl")) +
		para(run("Cavablar:", propBold)) +
		para(run("1-A, 2-A")) +
		para(run("3) B")) +
		para(run("4 - E"))

	result := parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{1, "Azərbaycanın paytaxtı?", []string{"*A:Bakı", "B:Gəncə"}},
		// The key wins over the author's bold mark and the conflict is shown.
		{2, "Ən uzun çay?", []string{"*A:Kür", "B:Araz"}},
		{3, "Ən hündür dağ?", []string{"A:Tufandağ", "*B:Bazardüzü"}},
		{4, "Rəng?", []string{"A:Qırmızı", "B:Yaşıl"}},
	})
	checkIssues(t, result.Issues, map[string][]int{
		IssueAnswerKeyConflict: {2},
		IssueAnswerKeyInvalid:  {4},
	})
}

func TestAnswerKeyTableAndHeadlessKeyLine(t *testing.T) {
	cell := func(text string) string { return "<w:tc>" + para(run(text)) + "</w:tc>" }
	row := func(left, right string) string { return "<w:tr>" + cell(left) + cell(right) + "</w:tr>" }
	questions := para(run("1. Bir?")) + para(run("A) a1")) + para(run("B) b1")) +
		para(run("2. İki?")) + para(run("A) a2")) + para(run("B) b2")) +
		para(run("3. Üç?")) + para(run("A) a3")) + para(run("B) b3"))

	table := para(run("Answer key")) +
		"<w:tbl>" + row("Sual", "Cavab") + row("1", "B") + row("2", "A") + row("3", "B") + "</w:tbl>"
	result := parseDocx(t, questions+table, "", "")
	checkQuestions(t, result.Questions, []want{
		{1, "Bir?", []string{"A:a1", "*B:b1"}},
		{2, "İki?", []string{"*A:a2", "B:b2"}},
		{3, "Üç?", []string{"A:a3", "*B:b3"}},
	})
	checkIssues(t, result.Issues, map[string][]int{})

	// Without a heading a line needs at least three entries in capitals.
	result = parseDocx(t, questions+para(run("1-B  2-B  3-A")), "", "")
	checkQuestions(t, result.Questions, []want{
		{1, "Bir?", []string{"A:a1", "*B:b1"}},
		{2, "İki?", []string{"A:a2", "*B:b2"}},
		{3, "Üç?", []string{"*A:a3", "B:b3"}},
	})
}

func TestMatchingOptionsAreNotAnAnswerKey(t *testing.T) {
	numbering := numberingXML(list{1, []listLevel{decimalParen}}, list{2, []listLevel{lowerParen}})
	body := listPara(1, 0, run("Match the words")) +
		listPara(2, 0, run("1-a ,2-b, 3-c", propBold)) +
		listPara(2, 0, run("1-b, 2-a, 3-c")) +
		listPara(2, 0, run("1-c, 2-b, 3-a"))

	result := parseDocx(t, body, numbering, "")
	checkQuestions(t, result.Questions, []want{
		{1, "Match the words", []string{"*A:1-a ,2-b, 3-c", "B:1-b, 2-a, 3-c", "C:1-c, 2-b, 3-a"}},
	})
	checkIssues(t, result.Issues, map[string][]int{})
}

func TestQuestionKeyLine(t *testing.T) {
	body := para(run("1. Sual bir?")) +
		para(run("A) x")) + para(run("B) y")) + para(run("C) z")) +
		para(run("Cavab: C")) +
		para(run("2. Sual iki?")) +
		para(run("A) x")) + para(run("B) y")) +
		para(run("Düzgün cavab - (b)"))

	result := parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{1, "Sual bir?", []string{"A:x", "B:y", "*C:z"}},
		{2, "Sual iki?", []string{"A:x", "*B:y"}},
	})
	checkIssues(t, result.Issues, map[string][]int{})
}

func TestDocumentSaysAIsAlwaysCorrect(t *testing.T) {
	body := para(run("Qeyd: bütün suallarda düzgün cavab A variantıdır.")) +
		unmarkedQuestions()

	result := parseDocx(t, body, "", "")
	checkQuestions(t, result.Questions, []want{
		{1, "Azərbaycanın paytaxtı?", []string{"*A:Bakı", "B:Gəncə", "C:Şəki"}},
		{2, "Ən uzun çay?", []string{"*A:Kür", "B:Araz"}},
		{3, "Ən hündür dağ?", []string{"B:Tufandağ", "C:Bazardüzü"}},
		{4, "Rəngi seçin", []string{"A:Qırmızı", "A:Yaşıl"}},
	})
	checkIssues(t, result.Issues, map[string][]int{
		IssueAnswerKeyInvalid:       {3},
		IssueCorrectAnswerAmbiguous: {4},
		"duplicate_option_label":    {4},
	})

	for _, text := range []string{
		"Правильный ответ везде — A.",
		"The correct answer is always A",
		"Düzgün cavab hər yerdə A-dır",
	} {
		if !isAllAStatement(text) {
			t.Errorf("not recognised: %q", text)
		}
	}
	for _, text := range []string{
		"A correct answer gets one point in all questions",
		"Bütün sualları cavablandırın",
		"Hansı halda düzgün cavab alınır?",
	} {
		if isAllAStatement(text) {
			t.Errorf("false statement: %q", text)
		}
	}
}

func TestAnswerModeAllAWithoutMarks(t *testing.T) {
	result := parseDocx(t, unmarkedQuestions(), "", AnswerModeAllA)
	checkQuestions(t, result.Questions, []want{
		{1, "Azərbaycanın paytaxtı?", []string{"*A:Bakı", "B:Gəncə", "C:Şəki"}},
		{2, "Ən uzun çay?", []string{"*A:Kür", "B:Araz"}},
		{3, "Ən hündür dağ?", []string{"B:Tufandağ", "C:Bazardüzü"}},
		{4, "Rəngi seçin", []string{"A:Qırmızı", "A:Yaşıl"}},
	})
	checkIssues(t, result.Issues, map[string][]int{
		IssueAnswerModeLabelMissing:  {3},
		IssueAnswerModeLabelRepeated: {4},
		"duplicate_option_label":     {4},
	})
}

func TestAnswerModeAllAReportsConflicts(t *testing.T) {
	numbering := numberingXML(list{1, []listLevel{decimalDot}}, list{2, []listLevel{upperParen}}, list{3, []listLevel{upperParen}})
	body := listPara(1, 0, run("Bir?")) +
		listPara(2, 0, run("a")) + listPara(2, 0, run("b", propBold)) +
		listPara(1, 0, run("İki?")) +
		listPara(3, 0, run("a")) + listPara(3, 0, run("b")) +
		para(run("Cavab: B"))

	result := parseDocx(t, body, numbering, AnswerModeAllA)
	checkQuestions(t, result.Questions, []want{
		{1, "Bir?", []string{"*A:a", "B:b"}},
		{2, "İki?", []string{"*A:a", "B:b"}},
	})
	checkIssues(t, result.Issues, map[string][]int{IssueAnswerKeyConflict: {1, 2}})
}

func TestNoKeyAndNoModeLeavesOptionsUnmarked(t *testing.T) {
	result := parseDocx(t, unmarkedQuestions(), "", "")
	for _, question := range result.Questions {
		for _, option := range question.Options {
			if option.IsCorrect {
				t.Fatalf("question %d option %s is marked", question.SourceNumber, option.Label)
			}
		}
	}
	checkIssues(t, result.Issues, map[string][]int{
		IssueCorrectAnswerMissing: {1, 2, 3, 4},
		"duplicate_option_label":  {4},
	})
	for _, issue := range result.Issues {
		if issue.Code == "exactly_one_correct_option" {
			t.Fatal("validator issue duplicates the parser issue")
		}
	}
}
