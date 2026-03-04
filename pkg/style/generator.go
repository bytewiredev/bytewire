package style

import (
	"fmt"
	"strings"
)

// cssRules maps class names to their CSS definitions.
var cssRules = map[Class]string{
	Flex:           "display:flex",
	FlexCol:        "flex-direction:column",
	FlexRow:        "flex-direction:row",
	FlexWrap:       "flex-wrap:wrap",
	ItemsCenter:    "align-items:center",
	JustifyCenter:  "justify-content:center",
	JustifyBetween: "justify-content:space-between",
	Gap1:           "gap:0.25rem",
	Gap2:           "gap:0.5rem",
	Gap4:           "gap:1rem",
	Gap8:           "gap:2rem",
	P1:             "padding:0.25rem",
	P2:             "padding:0.5rem",
	P4:             "padding:1rem",
	P8:             "padding:2rem",
	M1:             "margin:0.25rem",
	M2:             "margin:0.5rem",
	M4:             "margin:1rem",
	M8:             "margin:2rem",
	Mx2:            "margin-left:0.5rem;margin-right:0.5rem",
	My2:            "margin-top:0.5rem;margin-bottom:0.5rem",
	Px4:            "padding-left:1rem;padding-right:1rem",
	Py2:            "padding-top:0.5rem;padding-bottom:0.5rem",
	TextSm:         "font-size:0.875rem;line-height:1.25rem",
	TextBase:       "font-size:1rem;line-height:1.5rem",
	TextLg:         "font-size:1.125rem;line-height:1.75rem",
	TextXl:         "font-size:1.25rem;line-height:1.75rem",
	Text2Xl:        "font-size:1.5rem;line-height:2rem",
	Text3Xl:        "font-size:1.875rem;line-height:2.25rem",
	FontBold:       "font-weight:700",
	FontMedium:     "font-weight:500",
	TextCenter:     "text-align:center",
	BgWhite:        "background-color:#ffffff",
	BgGray100:      "background-color:#f3f4f6",
	BgGray800:      "background-color:#1f2937",
	BgBlue500:      "background-color:#3b82f6",
	BgBlue600:      "background-color:#2563eb",
	BgGreen500:     "background-color:#22c55e",
	BgRed500:       "background-color:#ef4444",
	TextWhite:      "color:#ffffff",
	TextGray700:    "color:#374151",
	TextGray900:    "color:#111827",
	TextBlue500:    "color:#3b82f6",
	Rounded:        "border-radius:0.25rem",
	RoundedMd:      "border-radius:0.375rem",
	RoundedLg:      "border-radius:0.5rem",
	RoundedFull:    "border-radius:9999px",
	Border:         "border-width:1px",
	BorderGray300:  "border-color:#d1d5db",
	Shadow:         "box-shadow:0 1px 3px rgba(0,0,0,0.1)",
	ShadowMd:       "box-shadow:0 4px 6px rgba(0,0,0,0.1)",
	ShadowLg:       "box-shadow:0 10px 15px rgba(0,0,0,0.1)",
	WFull:          "width:100%",
	HFull:          "height:100%",
	WScreen:        "width:100vw",
	HScreen:        "height:100vh",
	MaxWLg:         "max-width:32rem",
	MaxWXl:         "max-width:36rem",
	MaxW2Xl:        "max-width:42rem",
	W4:             "width:1rem",
	H4:             "height:1rem",
	W6:             "width:1.5rem",
	H6:             "height:1.5rem",
	W8:             "width:2rem",
	H8:             "height:2rem",
	MinW0:          "min-width:0",
	Fixed:          "position:fixed",
	Absolute:       "position:absolute",
	Relative:       "position:relative",
	Inset0:         "inset:0",
	Top0:           "top:0",
	Right0:         "right:0",
	Z10:            "z-index:10",
	Z50:            "z-index:50",
	Hidden:         "display:none",
	Block:          "display:block",
	Inline:         "display:inline-block",
	Overflow:       "overflow:auto",
	CursorPointer:  "cursor:pointer",
	BgBlack:        "background-color:#000000",
	BgYellow500:    "background-color:#eab308",
	BgYellow100:    "background-color:#fef9c3",
	BgGreen100:     "background-color:#dcfce7",
	BgRed100:       "background-color:#fee2e2",
	BgBlue100:      "background-color:#dbeafe",
	BgOpacity50:    "background-color:rgba(0,0,0,0.5)",
	TextYellow800:  "color:#854d0e",
	TextGreen800:   "color:#166534",
	TextRed800:     "color:#991b1b",
	TextRed500:     "color:#ef4444",
	TextGray500:    "color:#6b7280",
	TextGray600:    "color:#4b5563",
	P0:             "padding:0",
	P3:             "padding:0.75rem",
	P6:             "padding:1.5rem",
	Px2:            "padding-left:0.5rem;padding-right:0.5rem",
	Px3:            "padding-left:0.75rem;padding-right:0.75rem",
	Py1:            "padding-top:0.25rem;padding-bottom:0.25rem",
	Py3:            "padding-top:0.75rem;padding-bottom:0.75rem",
	Mt2:            "margin-top:0.5rem",
	Mb2:            "margin-bottom:0.5rem",
	Mb4:            "margin-bottom:1rem",
	Ml2:            "margin-left:0.5rem",
	Mr2:            "margin-right:0.5rem",
	BorderL4:       "border-left-width:4px",
	BorderGreen500: "border-color:#22c55e",
	BorderRed500:   "border-color:#ef4444",
	BorderYellow500: "border-color:#eab308",
	BorderBlue500:  "border-color:#3b82f6",
	AnimateSpin:    "animation:spin 1s linear infinite",
}

// Generate produces a minimal CSS stylesheet containing only the classes used.
// This is Bytewire's dead-code eliminating CSS compiler.
func Generate(used []Class) string {
	var sb strings.Builder
	sb.WriteString("/* Bytewire Generated Stylesheet - Zero Dead Code */\n")

	for _, cls := range used {
		if rule, ok := cssRules[cls]; ok {
			fmt.Fprintf(&sb, ".%s{%s}\n", cls, rule)
		}
	}

	return sb.String()
}

// GenerateAll produces a stylesheet with every registered class.
func GenerateAll() string {
	all := make([]Class, 0, len(cssRules))
	for cls := range cssRules {
		all = append(all, cls)
	}
	return Generate(all)
}
