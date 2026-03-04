// Package components provides reusable UI components built on Bytewire's dom and style packages.
package components

import (
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/style"
)

// TextInput creates a styled text input with two-way binding to a string signal.
func TextInput(placeholder string, value *dom.Signal[string]) *dom.Node {
	return dom.Input(
		dom.Attr("type", "text"),
		dom.Attr("placeholder", placeholder),
		dom.Attr("value", value.Get()),
		dom.Class(style.Classes(
			style.WFull, style.Px3, style.Py2,
			style.Border, style.BorderGray300, style.RoundedMd,
			style.TextBase, style.TextGray900,
		)),
		dom.OnInput(func(data []byte) {
			value.Set(string(data))
		}),
	)
}

// Checkbox creates a styled checkbox with a label, bound to a bool signal.
func Checkbox(label string, checked *dom.Signal[bool]) *dom.Node {
	return dom.Label(
		dom.Class(style.Classes(style.Flex, style.ItemsCenter, style.Gap2, style.CursorPointer)),
		dom.Children(
			dom.Input(
				dom.Attr("type", "checkbox"),
				dom.AttrF(checked, "checked", func(v bool) string {
					if v {
						return "true"
					}
					return ""
				}),
				dom.Class(style.Classes(style.W4, style.H4)),
				dom.OnClick(func(_ []byte) {
					checked.Update(func(v bool) bool { return !v })
				}),
			),
			dom.Span(
				dom.Class(style.Classes(style.TextBase, style.TextGray700)),
				dom.Children(dom.Text(label)),
			),
		),
	)
}

// Select creates a styled dropdown select bound to a string signal.
func Select(options []string, selected *dom.Signal[string]) *dom.Node {
	opts := make([]*dom.Node, len(options))
	for i, opt := range options {
		opts[i] = dom.OptionEl(
			dom.Attr("value", opt),
			dom.Children(dom.Text(opt)),
		)
	}

	return dom.SelectEl(
		dom.Attr("value", selected.Get()),
		dom.Class(style.Classes(
			style.WFull, style.Px3, style.Py2,
			style.Border, style.BorderGray300, style.RoundedMd,
			style.TextBase, style.TextGray900, style.BgWhite,
		)),
		dom.Children(opts...),
		dom.OnInput(func(data []byte) {
			selected.Set(string(data))
		}),
	)
}

// TextArea creates a styled textarea with two-way binding to a string signal.
func TextArea(placeholder string, value *dom.Signal[string]) *dom.Node {
	return dom.Textarea(
		dom.Attr("placeholder", placeholder),
		dom.Class(style.Classes(
			style.WFull, style.Px3, style.Py2,
			style.Border, style.BorderGray300, style.RoundedMd,
			style.TextBase, style.TextGray900,
		)),
		dom.Children(dom.Text(value.Get())),
		dom.OnInput(func(data []byte) {
			value.Set(string(data))
		}),
	)
}
