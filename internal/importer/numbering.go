package importer

import (
	"strconv"
	"strings"
)

type xmlLevel struct {
	Ilvl       string    `xml:"ilvl,attr"`
	Start      *xmlValue `xml:"start"`
	NumFmt     *xmlValue `xml:"numFmt"`
	LvlText    *xmlValue `xml:"lvlText"`
	LvlRestart *xmlValue `xml:"lvlRestart"`
}

type xmlAbstractNum struct {
	ID           string     `xml:"abstractNumId,attr"`
	Levels       []xmlLevel `xml:"lvl"`
	NumStyleLink *xmlValue  `xml:"numStyleLink"`
}

type xmlNum struct {
	ID         string    `xml:"numId,attr"`
	AbstractID *xmlValue `xml:"abstractNumId"`
	Overrides  []struct {
		Ilvl          string    `xml:"ilvl,attr"`
		StartOverride *xmlValue `xml:"startOverride"`
		Level         *xmlLevel `xml:"lvl"`
	} `xml:"lvlOverride"`
}

type xmlNumbering struct {
	Abstracts []xmlAbstractNum `xml:"abstractNum"`
	Nums      []xmlNum         `xml:"num"`
}

type numberingLevel struct {
	start   int
	format  string
	text    string
	restart int // -1: restart after any higher level, 0: never, n: after level n-1
}

type numberingInstance struct {
	abstractID     string
	levels         map[int]numberingLevel
	startOverrides map[int]int
}

// numberingState renders automatic list numbers the way Word does: counters
// belong to the abstract list, a w:num with startOverride restarts its
// levels the first time it is used, and a level restarts after a higher
// level is used.
type numberingState struct {
	instances map[string]*numberingInstance
	counters  map[string]map[int]int
	used      map[string]bool
}

func parseLevel(level xmlLevel) numberingLevel {
	result := numberingLevel{start: 1, format: "decimal", restart: -1}

	if level.Start != nil {
		if value, err := strconv.Atoi(strings.TrimSpace(level.Start.Val)); err == nil {
			result.start = value
		}
	}
	if level.NumFmt != nil && level.NumFmt.Val != "" {
		result.format = level.NumFmt.Val
	}
	if level.LvlText != nil {
		result.text = level.LvlText.Val
	}
	if level.LvlRestart != nil {
		if value, err := strconv.Atoi(strings.TrimSpace(level.LvlRestart.Val)); err == nil {
			result.restart = value
		}
	}

	return result
}

func newNumberingState(document *xmlNumbering, styles *styleSheet) *numberingState {
	state := &numberingState{
		instances: make(map[string]*numberingInstance),
		counters:  make(map[string]map[int]int),
		used:      make(map[string]bool),
	}

	if document == nil {
		return state
	}

	abstracts := make(map[string]xmlAbstractNum, len(document.Abstracts))
	for _, abstract := range document.Abstracts {
		abstracts[abstract.ID] = abstract
	}

	numAbstract := make(map[string]string, len(document.Nums))
	for _, num := range document.Nums {
		if num.AbstractID != nil {
			numAbstract[num.ID] = num.AbstractID.Val
		}
	}

	// An abstract list may only point at a numbering style that in turn
	// references the list that holds the real level definitions.
	resolveAbstract := func(id string) (string, xmlAbstractNum, bool) {
		for range 4 {
			abstract, ok := abstracts[id]
			if !ok {
				return "", xmlAbstractNum{}, false
			}
			if abstract.NumStyleLink == nil || len(abstract.Levels) > 0 {
				return id, abstract, true
			}
			numbering := styles.numbering(abstract.NumStyleLink.Val)
			if numbering == nil || numbering.NumID == nil {
				return id, abstract, true
			}
			next, ok := numAbstract[numbering.NumID.Val]
			if !ok || next == id {
				return id, abstract, true
			}
			id = next
		}
		return "", xmlAbstractNum{}, false
	}

	for _, num := range document.Nums {
		if num.AbstractID == nil {
			continue
		}
		abstractID, abstract, ok := resolveAbstract(num.AbstractID.Val)
		if !ok {
			continue
		}

		instance := &numberingInstance{
			abstractID:     abstractID,
			levels:         make(map[int]numberingLevel),
			startOverrides: make(map[int]int),
		}
		for _, level := range abstract.Levels {
			if index, err := strconv.Atoi(level.Ilvl); err == nil {
				instance.levels[index] = parseLevel(level)
			}
		}
		for _, override := range num.Overrides {
			index, err := strconv.Atoi(override.Ilvl)
			if err != nil {
				continue
			}
			if override.Level != nil {
				instance.levels[index] = parseLevel(*override.Level)
			}
			if override.StartOverride != nil {
				if value, err := strconv.Atoi(strings.TrimSpace(override.StartOverride.Val)); err == nil {
					instance.startOverrides[index] = value
				}
			}
		}

		state.instances[num.ID] = instance
	}

	return state
}

func listKind(format string) ListKind {
	switch format {
	case "decimal", "decimalZero", "decimalEnclosedParen",
		"decimalEnclosedFullstop", "decimalFullWidth", "decimalHalfWidth":
		return ListNumber
	case "lowerLetter", "upperLetter":
		return ListLetter
	case "bullet", "none", "":
		return ListNone
	default:
		return ListOther
	}
}

// next advances the counter of a numbered paragraph and returns its label.
func (state *numberingState) next(numID string, ilvl int) (string, ListKind, int, bool) {
	instance, ok := state.instances[numID]
	if !ok || numID == "" || numID == "0" {
		return "", ListNone, 0, false
	}

	level, ok := instance.levels[ilvl]
	if !ok {
		return "", ListNone, 0, false
	}

	counters := state.counters[instance.abstractID]
	if counters == nil {
		counters = make(map[int]int)
		state.counters[instance.abstractID] = counters
	}

	if !state.used[numID] {
		state.used[numID] = true
		for index, start := range instance.startOverrides {
			counters[index] = start - 1
		}
	}

	if value, ok := counters[ilvl]; ok {
		counters[ilvl] = value + 1
	} else {
		counters[ilvl] = level.start
	}

	for index := range counters {
		if index <= ilvl {
			continue
		}
		child := instance.levels[index]
		if child.restart == 0 {
			continue
		}
		if child.restart > 0 && ilvl > child.restart-1 {
			continue
		}
		delete(counters, index)
	}

	var label strings.Builder
	text := level.text
	for index := 0; index < len(text); index++ {
		if text[index] == '%' && index+1 < len(text) &&
			text[index+1] >= '1' && text[index+1] <= '9' {
			referenced := int(text[index+1] - '1')
			referencedLevel, exists := instance.levels[referenced]
			value, counted := counters[referenced]
			if !counted {
				value = 1
				if exists {
					value = referencedLevel.start
				}
			}
			format := "decimal"
			if exists {
				format = referencedLevel.format
			}
			label.WriteString(formatNumber(value, format))
			index++
			continue
		}
		label.WriteByte(text[index])
	}

	return label.String(), listKind(level.format), counters[ilvl], true
}

func formatNumber(value int, format string) string {
	switch format {
	case "lowerLetter", "upperLetter":
		if value < 1 {
			value = 1
		}
		letter := byte('a' + (value-1)%26)
		if format == "upperLetter" {
			letter = byte('A' + (value-1)%26)
		}
		return strings.Repeat(string(letter), (value-1)/26+1)
	case "lowerRoman":
		return strings.ToLower(romanNumeral(value))
	case "upperRoman":
		return romanNumeral(value)
	case "decimalZero":
		if value < 10 && value >= 0 {
			return "0" + strconv.Itoa(value)
		}
		return strconv.Itoa(value)
	case "bullet", "none":
		return ""
	default:
		return strconv.Itoa(value)
	}
}

func romanNumeral(value int) string {
	if value <= 0 || value >= 4000 {
		return strconv.Itoa(value)
	}

	numerals := []struct {
		value  int
		symbol string
	}{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}

	var result strings.Builder
	for _, numeral := range numerals {
		for value >= numeral.value {
			result.WriteString(numeral.symbol)
			value -= numeral.value
		}
	}
	return result.String()
}
