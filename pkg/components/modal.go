package components

import (
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/style"
)

// Modal creates a modal dialog controlled by a bool signal.
// When visible is true, a backdrop overlay and centered dialog are shown.
func Modal(title string, visible *dom.Signal[bool], children ...*dom.Node) *dom.Node {
	closeBtn := dom.Button(
		dom.Class(style.Classes(style.TextGray500, style.TextLg, style.FontBold, style.CursorPointer)),
		dom.Children(dom.Text("\u00d7")),
		dom.OnClick(func(_ []byte) {
			visible.Set(false)
		}),
	)

	header := dom.Div(
		dom.Class(style.Classes(style.Flex, style.JustifyBetween, style.ItemsCenter, style.Mb4)),
		dom.Children(
			dom.H2(
				dom.Class(style.Classes(style.TextXl, style.FontBold, style.TextGray900)),
				dom.Children(dom.Text(title)),
			),
			closeBtn,
		),
	)

	body := dom.Div(dom.Children(children...))

	dialog := dom.Div(
		dom.Class(style.Classes(
			style.BgWhite, style.RoundedLg, style.ShadowLg, style.P6,
			style.MaxWLg, style.WFull, style.Relative,
		)),
		dom.Children(header, body),
	)

	backdrop := dom.Div(
		dom.Class(style.Classes(
			style.Fixed, style.Inset0, style.Z50,
			style.Flex, style.ItemsCenter, style.JustifyCenter,
			style.BgOpacity50,
		)),
		dom.Children(dialog),
		dom.OnClick(func(_ []byte) {
			visible.Set(false)
		}),
	)

	return dom.If(visible, func(v bool) bool { return v },
		func() *dom.Node { return backdrop },
		nil,
	)
}
