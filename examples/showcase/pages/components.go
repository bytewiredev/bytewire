package pages

import (
	"fmt"

	"github.com/bytewiredev/bytewire/pkg/components"
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/engine"
	"github.com/bytewiredev/bytewire/pkg/style"
)

// Components renders a gallery showcasing all available UI components.
func Components(s *engine.Session) *dom.Node {
	return dom.Div(
		dom.Children(
			dom.H2(
				dom.Class(style.Classes(style.Text2Xl, style.FontBold, style.TextGray900, style.Mb6)),
				dom.Children(dom.Text("Component Gallery")),
			),
			formsSection(s),
			displaySection(),
			dataSection(s),
			feedbackSection(s),
		),
	)
}

func sectionTitle(title string) *dom.Node {
	return dom.H3(
		dom.Class(style.Classes(style.TextXl, style.FontBold, style.TextGray900, style.Mb4)),
		dom.Children(dom.Text(title)),
	)
}

func formsSection(s *engine.Session) *dom.Node {
	inputVal := dom.NewSignal("")
	checkVal := dom.NewSignal(false)
	selectVal := dom.NewSignal("option1")

	return dom.Div(
		dom.Class(style.Classes(style.Mb6)),
		dom.Children(
			sectionTitle("Forms"),
			dom.Div(
				dom.Class(style.Classes(style.Grid, style.GridCols2, style.Gap6)),
				dom.Children(
					// Text Input with live preview
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.Label(
								dom.Class(style.Classes(style.Block, style.TextSm, style.FontMedium, style.TextGray700, style.Mb2)),
								dom.Children(dom.Text("Text Input")),
							),
							dom.Input(
								dom.Class(style.Classes(style.WFull, style.Border, style.BorderGray300, style.RoundedMd, style.Px3, style.Py2)),
								dom.Attr("type", "text"),
								dom.Attr("placeholder", "Type something..."),
								dom.OnInput(func(data []byte) {
									inputVal.Set(string(data))
								}),
							),
							dom.P(
								dom.Class(style.Classes(style.Mt2, style.TextSm, style.TextGray500)),
								dom.Children(
									dom.Text("Preview: "),
									dom.TextF(inputVal, func(v string) string {
										if v == "" {
											return "(empty)"
										}
										return v
									}),
								),
							),
						),
					),

					// Checkbox
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.Label(
								dom.Class(style.Classes(style.Flex, style.ItemsCenter, style.Gap2)),
								dom.Children(
									dom.Input(
										dom.Attr("type", "checkbox"),
										dom.OnClick(func(_ []byte) {
											checkVal.Update(func(v bool) bool { return !v })
										}),
									),
									dom.Span(
										dom.Class(style.Classes(style.TextSm, style.TextGray700)),
										dom.Children(dom.Text("Enable feature")),
									),
								),
							),
							dom.P(
								dom.Class(style.Classes(style.Mt2, style.TextSm, style.TextGray500)),
								dom.Children(
									dom.TextF(checkVal, func(v bool) string {
										if v {
											return "Enabled"
										}
										return "Disabled"
									}),
								),
							),
						),
					),

					// Select
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.Label(
								dom.Class(style.Classes(style.Block, style.TextSm, style.FontMedium, style.TextGray700, style.Mb2)),
								dom.Children(dom.Text("Select")),
							),
							dom.SelectEl(
								dom.Class(style.Classes(style.WFull, style.Border, style.BorderGray300, style.RoundedMd, style.Px3, style.Py2)),
								dom.OnInput(func(data []byte) {
									selectVal.Set(string(data))
								}),
								dom.Children(
									dom.OptionEl(dom.Attr("value", "option1"), dom.Children(dom.Text("Option 1"))),
									dom.OptionEl(dom.Attr("value", "option2"), dom.Children(dom.Text("Option 2"))),
									dom.OptionEl(dom.Attr("value", "option3"), dom.Children(dom.Text("Option 3"))),
								),
							),
							dom.P(
								dom.Class(style.Classes(style.Mt2, style.TextSm, style.TextGray500)),
								dom.Children(
									dom.Text("Selected: "),
									dom.TextF(selectVal, func(v string) string { return v }),
								),
							),
						),
					),

					// Textarea
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.Label(
								dom.Class(style.Classes(style.Block, style.TextSm, style.FontMedium, style.TextGray700, style.Mb2)),
								dom.Children(dom.Text("Textarea")),
							),
							dom.Textarea(
								dom.Class(style.Classes(style.WFull, style.Border, style.BorderGray300, style.RoundedMd, style.Px3, style.Py2)),
								dom.Attr("rows", "3"),
								dom.Attr("placeholder", "Enter long text..."),
							),
						),
					),
				),
			),
		),
	)
}

func displaySection() *dom.Node {
	badgeVariants := []struct {
		label string
		bg    style.Class
		text  style.Class
	}{
		{"Success", style.BgGreen100, style.TextGreen800},
		{"Warning", style.BgYellow100, style.TextYellow800},
		{"Error", style.BgRed100, style.TextRed800},
		{"Info", style.BgBlue100, style.TextBlue500},
	}

	badges := make([]*dom.Node, 0, len(badgeVariants))
	for _, bv := range badgeVariants {
		badges = append(badges, dom.Span(
			dom.Class(style.Classes(bv.bg, bv.text, style.Px3, style.Py1, style.RoundedFull, style.TextSm, style.FontMedium)),
			dom.Children(dom.Text(bv.label)),
		))
	}

	alertVariants := []struct {
		label  string
		border style.Class
		bg     style.Class
		text   style.Class
	}{
		{"Operation completed successfully.", style.BorderGreen500, style.BgGreen100, style.TextGreen800},
		{"Please review the changes carefully.", style.BorderYellow500, style.BgYellow100, style.TextYellow800},
		{"An error occurred during processing.", style.BorderRed500, style.BgRed100, style.TextRed800},
	}

	alerts := make([]*dom.Node, 0, len(alertVariants))
	for _, av := range alertVariants {
		alerts = append(alerts, dom.Div(
			dom.Class(style.Classes(av.border, av.bg, av.text, style.BorderL4, style.P4, style.RoundedMd)),
			dom.Children(dom.Text(av.label)),
		))
	}

	return dom.Div(
		dom.Class(style.Classes(style.Mb6)),
		dom.Children(
			sectionTitle("Display"),
			dom.Div(
				dom.Class(style.Classes(style.Grid, style.GridCols2, style.Gap6)),
				dom.Children(
					// Badges
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.H3(
								dom.Class(style.Classes(style.FontBold, style.TextGray700, style.Mb4)),
								dom.Children(dom.Text("Badges")),
							),
							dom.Div(
								dom.Class(style.Classes(style.Flex, style.FlexWrap, style.Gap2)),
								dom.Children(badges...),
							),
						),
					),
					// Alerts
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.H3(
								dom.Class(style.Classes(style.FontBold, style.TextGray700, style.Mb4)),
								dom.Children(dom.Text("Alerts")),
							),
							dom.Div(
								dom.Class(style.Classes(style.FlexCol, style.Flex, style.Gap2)),
								dom.Children(alerts...),
							),
						),
					),
				),
			),
		),
	)
}

type tableRow struct {
	Name  string
	Email string
	Role  string
}

func dataSection(s *engine.Session) *dom.Node {
	counter := dom.NewSignal(3)
	rows := dom.NewListSignal([]tableRow{
		{"Alice", "alice@example.com", "Admin"},
		{"Bob", "bob@example.com", "Editor"},
		{"Charlie", "charlie@example.com", "Viewer"},
	})

	return dom.Div(
		dom.Class(style.Classes(style.Mb6)),
		dom.Children(
			sectionTitle("Data"),
			dom.Div(
				dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
				dom.Children(
					dom.Div(
						dom.Class(style.Classes(style.Flex, style.JustifyBetween, style.ItemsCenter, style.Mb4)),
						dom.Children(
							dom.H3(
								dom.Class(style.Classes(style.FontBold, style.TextGray700)),
								dom.Children(dom.Text("Users Table")),
							),
							dom.Div(
								dom.Class(style.Classes(style.Flex, style.Gap2)),
								dom.Children(
									dom.Button(
										dom.Class(style.Classes(
											style.BgGreen500, style.TextWhite, style.Px3, style.Py1,
											style.RoundedMd, style.TextSm,
										)),
										dom.Children(dom.Text("Add Row")),
										dom.OnClick(func(_ []byte) {
											n := counter.Get() + 1
											counter.Set(n)
											rows.Append(tableRow{
												Name:  fmt.Sprintf("User %d", n),
												Email: fmt.Sprintf("user%d@example.com", n),
												Role:  "Viewer",
											})
										}),
									),
									dom.Button(
										dom.Class(style.Classes(
											style.BgRed500, style.TextWhite, style.Px3, style.Py1,
											style.RoundedMd, style.TextSm,
										)),
										dom.Children(dom.Text("Remove Last")),
										dom.OnClick(func(_ []byte) {
											rows.Update(func(items []tableRow) []tableRow {
												if len(items) > 0 {
													return items[:len(items)-1]
												}
												return items
											})
										}),
									),
								),
							),
						),
					),
					dom.Table(
						dom.Class(style.Classes(style.WFull, style.TextSm)),
						dom.Children(
							dom.Thead(dom.Children(
								dom.Tr(dom.Children(
									dom.Th(dom.Class(style.Classes(style.Py2, style.Px3, style.TextGray500, style.FontMedium, style.BorderB)), dom.Children(dom.Text("Name"))),
									dom.Th(dom.Class(style.Classes(style.Py2, style.Px3, style.TextGray500, style.FontMedium, style.BorderB)), dom.Children(dom.Text("Email"))),
									dom.Th(dom.Class(style.Classes(style.Py2, style.Px3, style.TextGray500, style.FontMedium, style.BorderB)), dom.Children(dom.Text("Role"))),
								)),
							)),
							dom.For(rows, func(r tableRow) string { return r.Email }, func(r tableRow) *dom.Node {
								return dom.Tr(dom.Children(
									dom.Td(dom.Class(style.Classes(style.Py2, style.Px3, style.BorderB)), dom.Children(dom.Text(r.Name))),
									dom.Td(dom.Class(style.Classes(style.Py2, style.Px3, style.BorderB, style.TextGray500)), dom.Children(dom.Text(r.Email))),
									dom.Td(dom.Class(style.Classes(style.Py2, style.Px3, style.BorderB)), dom.Children(dom.Text(r.Role))),
								))
							}),
						),
					),
				),
			),
		),
	)
}

func feedbackSection(s *engine.Session) *dom.Node {
	modalVisible := dom.NewSignal(false)

	return dom.Div(
		dom.Class(style.Classes(style.Mb6)),
		dom.Children(
			sectionTitle("Feedback"),
			dom.Div(
				dom.Class(style.Classes(style.Grid, style.GridCols2, style.Gap6)),
				dom.Children(
					// Modal
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.H3(
								dom.Class(style.Classes(style.FontBold, style.TextGray700, style.Mb4)),
								dom.Children(dom.Text("Modal")),
							),
							dom.Button(
								dom.Class(style.Classes(
									style.BgBlue500, style.TextWhite, style.Px4, style.Py2,
									style.RoundedMd, style.FontMedium,
								)),
								dom.Children(dom.Text("Open Modal")),
								dom.OnClick(func(_ []byte) {
									modalVisible.Set(true)
								}),
							),
							components.Modal("Example Modal", modalVisible,
								dom.P(dom.Children(dom.Text("This is a modal dialog component controlled by a signal."))),
							),
						),
					),

					// Spinner
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.H3(
								dom.Class(style.Classes(style.FontBold, style.TextGray700, style.Mb4)),
								dom.Children(dom.Text("Loading Spinner")),
							),
							dom.Div(
								dom.Class(style.Classes(style.Flex, style.ItemsCenter, style.Gap4)),
								dom.Children(
									dom.Div(
										dom.Class(style.Classes(style.W8, style.H8, style.Border, style.RoundedFull, style.AnimateSpin, style.BorderBlue500)),
									),
									dom.Span(
										dom.Class(style.Classes(style.TextSm, style.TextGray500)),
										dom.Children(dom.Text("Loading...")),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}

