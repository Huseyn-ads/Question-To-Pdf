package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalSamplesMatchReference compares every local sample document with a
// reference made from the original. The documents are not part of the
// repository: put them in local-docx-samples/ (or QTP_DOCX_SAMPLES) and the
// references in its expected/ folder as <name>.json:
//
//	[{"text": "...", "options": [{"label": "A", "text": "...", "is_correct": true}]}]
func TestLocalSamplesMatchReference(t *testing.T) {
	directory := os.Getenv("QTP_DOCX_SAMPLES")
	if directory == "" {
		directory = filepath.Join("..", "..", "local-docx-samples")
	}

	documents, _ := filepath.Glob(filepath.Join(directory, "*.docx"))
	if len(documents) == 0 {
		t.Skip("no local DOCX samples")
	}

	type option struct {
		Label     string `json:"label"`
		Text      string `json:"text"`
		IsCorrect bool   `json:"is_correct"`
	}
	type question struct {
		Text    string   `json:"text"`
		Options []option `json:"options"`
	}

	for _, document := range documents {
		name := strings.TrimSuffix(filepath.Base(document), ".docx")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(directory, "expected", name+".json"))
			if err != nil {
				t.Skipf("no reference: %v", err)
			}
			var expected []question
			if err := json.Unmarshal(data, &expected); err != nil {
				t.Fatal(err)
			}

			paragraphs, err := ExtractParagraphs(document)
			if err != nil {
				t.Fatal(err)
			}
			got := Parse(paragraphs, ParseOptions{}).Questions

			if len(got) != len(expected) {
				t.Fatalf("questions: got %d, want %d", len(got), len(expected))
			}
			for index := range expected {
				if got[index].Text != expected[index].Text {
					t.Errorf("question %d text:\n got  %q\n want %q", index+1, got[index].Text, expected[index].Text)
				}
				if len(got[index].Options) != len(expected[index].Options) {
					t.Errorf("question %d: got %d options, want %d", index+1, len(got[index].Options), len(expected[index].Options))
					continue
				}
				for optionIndex, want := range expected[index].Options {
					have := got[index].Options[optionIndex]
					if have.Label != want.Label || have.Text != want.Text || have.IsCorrect != want.IsCorrect {
						t.Errorf("question %d option %d: got %+v, want %+v", index+1, optionIndex+1, have, want)
					}
				}
			}
		})
	}
}
