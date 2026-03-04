package components

import (
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/style"
)

// Column defines a table column with a header and a cell renderer.
type Column[T any] struct {
	Header string
	Render func(T) *dom.Node
}

// Table creates a styled table driven by a ListSignal with keyed rows.
func Table[T any](rows *dom.ListSignal[T], keyFn func(T) string, columns []Column[T]) *dom.Node {
	// Build header row
	headers := make([]*dom.Node, len(columns))
	for i, col := range columns {
		headers[i] = dom.Th(
			dom.Class(style.Classes(style.Px4, style.Py2, style.TextSm, style.FontBold, style.TextGray700)),
			dom.Children(dom.Text(col.Header)),
		)
	}

	thead := dom.Thead(
		dom.Class(style.Classes(style.BgGray100)),
		dom.Children(
			dom.Tr(dom.Children(headers...)),
		),
	)

	// Build body using For to reactively render rows
	tbody := dom.For(rows, keyFn, func(item T) *dom.Node {
		cells := make([]*dom.Node, len(columns))
		for i, col := range columns {
			cells[i] = dom.Td(
				dom.Class(style.Classes(style.Px4, style.Py2, style.TextSm, style.TextGray900)),
				dom.Children(col.Render(item)),
			)
		}
		return dom.Tr(
			dom.Class(style.Classes(style.Border, style.BorderGray300)),
			dom.Children(cells...),
		)
	})

	return dom.Table(
		dom.Class(style.Classes(style.WFull, style.Border, style.BorderGray300, style.RoundedMd)),
		dom.Children(thead, tbody),
	)
}
