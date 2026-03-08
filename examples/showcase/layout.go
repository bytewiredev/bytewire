package main

import (
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/style"
)

// navItem defines a sidebar navigation entry.
type navItem struct {
	label string
	href  string
}

var navItems = []navItem{
	{"Home", "/"},
	{"Components", "/components"},
	{"Charts", "/charts"},
	{"About", "/about"},
}

// Layout wraps page content with a sidebar navigation and main content area.
func Layout(currentPath string, content *dom.Node) *dom.Node {
	// Sidebar
	navLinks := make([]*dom.Node, 0, len(navItems))
	for _, item := range navItems {
		linkClasses := style.Classes(
			style.Block, style.Px4, style.Py2, style.RoundedMd, style.TextSm,
		)
		if item.href == currentPath {
			linkClasses += " " + style.Classes(style.BgBlue600, style.TextWhite)
		} else {
			linkClasses += " " + style.Classes(style.TextGray300)
		}

		navLinks = append(navLinks, dom.A(
			dom.Class(linkClasses),
			dom.Link(item.href),
			dom.Children(dom.Text(item.label)),
		))
	}

	sidebar := dom.Nav(
		dom.Class(style.Classes(
			style.W64, style.MinHScreen, style.BgGray900, style.TextWhite,
			style.Shrink0, style.FlexCol, style.Flex, style.P4, style.Gap2,
		)),
		dom.Children(
			dom.Div(
				dom.Class(style.Classes(style.Px4, style.Py3, style.Mb6)),
				dom.Children(
					dom.H1(
						dom.Class(style.Classes(style.TextXl, style.FontBold, style.TextWhite)),
						dom.Children(dom.Text("Bytewire")),
					),
					dom.P(
						dom.Class(style.Classes(style.TextXs, style.TextGray400, style.Uppercase, style.TrackingWide)),
						dom.Children(dom.Text("Showcase")),
					),
				),
			),
			dom.Div(
				dom.Class(style.Classes(style.FlexCol, style.Flex, style.Gap1)),
				dom.Children(navLinks...),
			),
		),
	)

	main := dom.Main(
		dom.Class(style.Classes(style.Flex1, style.BgGray50, style.Px6, style.Py8, style.OverflowYAuto)),
		dom.Children(content),
	)

	return dom.Div(
		dom.Class(style.Classes(style.Flex, style.MinHScreen)),
		dom.Children(sidebar, main),
	)
}
