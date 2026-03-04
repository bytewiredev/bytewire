package components

import (
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/style"
)

// Card creates a styled card with a title and children.
func Card(title string, children ...*dom.Node) *dom.Node {
	header := dom.Div(
		dom.Class(style.Classes(style.Mb4)),
		dom.Children(
			dom.H3(
				dom.Class(style.Classes(style.TextLg, style.FontBold, style.TextGray900)),
				dom.Children(dom.Text(title)),
			),
		),
	)

	all := make([]*dom.Node, 0, 1+len(children))
	all = append(all, header)
	all = append(all, children...)

	return dom.Div(
		dom.Class(style.Classes(
			style.BgWhite, style.RoundedLg, style.ShadowMd, style.P6,
		)),
		dom.Children(all...),
	)
}

// Badge creates a small styled label. Variant should be "success", "error", or "warning".
func Badge(text string, variant string) *dom.Node {
	var bg, fg style.Class
	switch variant {
	case "success":
		bg = style.BgGreen100
		fg = style.TextGreen800
	case "error":
		bg = style.BgRed100
		fg = style.TextRed800
	case "warning":
		bg = style.BgYellow100
		fg = style.TextYellow800
	default:
		bg = style.BgGray100
		fg = style.TextGray700
	}

	return dom.Span(
		dom.Class(style.Classes(bg, fg, style.TextSm, style.FontMedium, style.Px2, style.Py1, style.RoundedFull)),
		dom.Children(dom.Text(text)),
	)
}

// Alert creates a styled alert message. AlertType should be "success", "error", "warning", or "info".
func Alert(message string, alertType string, dismissible bool) *dom.Node {
	var bg, border, fg style.Class
	switch alertType {
	case "success":
		bg = style.BgGreen100
		border = style.BorderGreen500
		fg = style.TextGreen800
	case "error":
		bg = style.BgRed100
		border = style.BorderRed500
		fg = style.TextRed800
	case "warning":
		bg = style.BgYellow100
		border = style.BorderYellow500
		fg = style.TextYellow800
	default: // info
		bg = style.BgBlue100
		border = style.BorderBlue500
		fg = style.TextGray700
	}

	content := dom.Span(
		dom.Class(style.Classes(fg)),
		dom.Children(dom.Text(message)),
	)

	if !dismissible {
		return dom.Div(
			dom.Class(style.Classes(bg, border, style.BorderL4, style.P4, style.RoundedMd)),
			dom.Children(content),
		)
	}

	visible := dom.NewSignal(true)

	closeBtn := dom.Button(
		dom.Class(style.Classes(fg, style.FontBold, style.Ml2, style.CursorPointer)),
		dom.Children(dom.Text("\u00d7")),
		dom.OnClick(func(_ []byte) {
			visible.Set(false)
		}),
	)

	return dom.If(visible, func(v bool) bool { return v },
		func() *dom.Node {
			return dom.Div(
				dom.Class(style.Classes(bg, border, style.BorderL4, style.P4, style.RoundedMd, style.Flex, style.JustifyBetween, style.ItemsCenter)),
				dom.Children(content, closeBtn),
			)
		},
		nil,
	)
}

// Spinner creates an animated loading spinner.
func Spinner() *dom.Node {
	return dom.Div(
		dom.Class(style.Classes(style.Flex, style.ItemsCenter, style.JustifyCenter)),
		dom.Children(
			dom.Div(
				dom.Class(style.Classes(style.W8, style.H8, style.Border, style.BorderBlue500, style.RoundedFull, style.AnimateSpin)),
				dom.Style("border-top-color", "transparent"),
			),
		),
	)
}
