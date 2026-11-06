package importer

import (
	"strconv"
	"strings"
)

// The structs below mirror the small part of WordprocessingML that the
// importer needs. encoding/xml matches attributes and elements by local
// name, so both transitional and strict namespaces are accepted.

type xmlValue struct {
	Val string `xml:"val,attr"`
}

type xmlShading struct {
	Fill string `xml:"fill,attr"`
	Val  string `xml:"val,attr"`
}

type xmlRunProperties struct {
	Bold      *xmlValue   `xml:"b"`
	Highlight *xmlValue   `xml:"highlight"`
	Color     *xmlValue   `xml:"color"`
	Underline *xmlValue   `xml:"u"`
	Shading   *xmlShading `xml:"shd"`
	Vanish    *xmlValue   `xml:"vanish"`
}

type xmlNumberingProperties struct {
	NumID *xmlValue `xml:"numId"`
	Ilvl  *xmlValue `xml:"ilvl"`
}

type xmlStyle struct {
	Type    string    `xml:"type,attr"`
	ID      string    `xml:"styleId,attr"`
	Default string    `xml:"default,attr"`
	BasedOn *xmlValue `xml:"basedOn"`
	PPr     *struct {
		NumPr *xmlNumberingProperties `xml:"numPr"`
	} `xml:"pPr"`
	RPr *xmlRunProperties `xml:"rPr"`
}

type xmlStyles struct {
	DocDefaults *struct {
		RPr *xmlRunProperties `xml:"rPrDefault>rPr"`
	} `xml:"docDefaults"`
	Styles []xmlStyle `xml:"style"`
}

// runFormat is a partially specified set of run properties. Nil means
// "inherit".
type runFormat struct {
	bold      *bool
	highlight *string
	color     *string
	underline *bool
	shading   *string
	vanish    *bool
}

func parseOnOff(value *xmlValue) *bool {
	if value == nil {
		return nil
	}

	result := true
	switch strings.ToLower(strings.TrimSpace(value.Val)) {
	case "0", "false", "off", "none":
		result = false
	}

	return &result
}

func stringPointer(value string) *string {
	return &value
}

func formatFromXML(properties *xmlRunProperties) runFormat {
	var format runFormat
	if properties == nil {
		return format
	}

	format.bold = parseOnOff(properties.Bold)
	format.vanish = parseOnOff(properties.Vanish)

	if properties.Highlight != nil {
		value := strings.TrimSpace(properties.Highlight.Val)
		if strings.EqualFold(value, "none") {
			value = ""
		}
		format.highlight = stringPointer(value)
	}

	if properties.Color != nil {
		format.color = stringPointer(colorClass(properties.Color.Val))
	}

	if properties.Underline != nil {
		value := strings.ToLower(strings.TrimSpace(properties.Underline.Val))
		underline := value != "none" && value != "0" && value != "false"
		format.underline = &underline
	}

	if properties.Shading != nil {
		format.shading = stringPointer(colorClass(properties.Shading.Fill))
	}

	return format
}

// over returns the format with unset fields taken from base.
func (format runFormat) over(base runFormat) runFormat {
	if format.bold == nil {
		format.bold = base.bold
	}
	if format.highlight == nil {
		format.highlight = base.highlight
	}
	if format.color == nil {
		format.color = base.color
	}
	if format.underline == nil {
		format.underline = base.underline
	}
	if format.shading == nil {
		format.shading = base.shading
	}
	if format.vanish == nil {
		format.vanish = base.vanish
	}

	return format
}

func (format runFormat) apply(run *TextRun) {
	run.Bold = format.bold != nil && *format.bold
	run.Underline = format.underline != nil && *format.underline
	if format.highlight != nil {
		run.Highlight = *format.highlight
	}
	if format.color != nil {
		run.Color = *format.color
	}
	if format.shading != nil {
		run.Shading = *format.shading
	}
}

type styleSheet struct {
	styles               map[string]xmlStyle
	defaults             runFormat
	defaultParagraphID   string
	resolvedRunFormats   map[string]runFormat
	resolvedNumberingPrs map[string]*xmlNumberingProperties
}

func newStyleSheet(document *xmlStyles) *styleSheet {
	sheet := &styleSheet{
		styles:               make(map[string]xmlStyle),
		resolvedRunFormats:   make(map[string]runFormat),
		resolvedNumberingPrs: make(map[string]*xmlNumberingProperties),
	}

	if document == nil {
		return sheet
	}

	if document.DocDefaults != nil {
		sheet.defaults = formatFromXML(document.DocDefaults.RPr)
	}

	for _, style := range document.Styles {
		if style.ID == "" {
			continue
		}
		sheet.styles[style.ID] = style
		if style.Type == "paragraph" && isTrue(style.Default) &&
			sheet.defaultParagraphID == "" {
			sheet.defaultParagraphID = style.ID
		}
	}

	return sheet
}

func isTrue(value string) bool {
	switch strings.ToLower(value) {
	case "1", "true", "on":
		return true
	}
	return false
}

// runFormat returns the run properties defined by a style and the styles it
// is based on.
func (sheet *styleSheet) runFormat(styleID string) runFormat {
	if styleID == "" {
		return runFormat{}
	}

	if cached, ok := sheet.resolvedRunFormats[styleID]; ok {
		return cached
	}

	var result runFormat
	seen := make(map[string]bool)
	for id := styleID; id != "" && !seen[id] && len(seen) < 32; {
		seen[id] = true
		style, ok := sheet.styles[id]
		if !ok {
			break
		}
		result = result.over(formatFromXML(style.RPr))
		id = ""
		if style.BasedOn != nil {
			id = style.BasedOn.Val
		}
	}

	sheet.resolvedRunFormats[styleID] = result
	return result
}

// numbering returns numbering properties inherited from a paragraph style.
func (sheet *styleSheet) numbering(styleID string) *xmlNumberingProperties {
	if styleID == "" {
		return nil
	}

	if cached, ok := sheet.resolvedNumberingPrs[styleID]; ok {
		return cached
	}

	var result *xmlNumberingProperties
	seen := make(map[string]bool)
	for id := styleID; id != "" && !seen[id] && len(seen) < 32; {
		seen[id] = true
		style, ok := sheet.styles[id]
		if !ok {
			break
		}
		if style.PPr != nil && style.PPr.NumPr != nil &&
			style.PPr.NumPr.NumID != nil {
			result = style.PPr.NumPr
			break
		}
		id = ""
		if style.BasedOn != nil {
			id = style.BasedOn.Val
		}
	}

	sheet.resolvedNumberingPrs[styleID] = result
	return result
}

// colorClass maps an RGB hex colour to a coarse hue name so that close
// shades (FF0000, EE0000, C00000) count as the same author mark. Black,
// grey, white and automatic colours return "".
func colorClass(hex string) string {
	hex = strings.TrimSpace(hex)
	if len(hex) != 6 {
		return ""
	}

	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return ""
	}

	red := float64((value>>16)&0xff) / 255
	green := float64((value>>8)&0xff) / 255
	blue := float64(value&0xff) / 255

	maximum := max(red, green, blue)
	minimum := min(red, green, blue)
	if maximum < 0.35 || maximum == 0 {
		return ""
	}
	saturation := (maximum - minimum) / maximum
	if saturation < 0.3 {
		return ""
	}

	var hue float64
	delta := maximum - minimum
	switch maximum {
	case red:
		hue = (green - blue) / delta
		if hue < 0 {
			hue += 6
		}
	case green:
		hue = (blue-red)/delta + 2
	default:
		hue = (red-green)/delta + 4
	}
	hue *= 60

	switch {
	case hue < 20:
		return "red"
	case hue < 45:
		return "orange"
	case hue < 70:
		return "yellow"
	case hue < 170:
		return "green"
	case hue < 200:
		return "cyan"
	case hue < 260:
		return "blue"
	case hue < 330:
		return "magenta"
	default:
		return "red"
	}
}
