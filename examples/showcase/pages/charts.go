package pages

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/bytewiredev/bytewire/pkg/components"
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/engine"
	"github.com/bytewiredev/bytewire/pkg/style"
)

// Charts renders interactive chart demos with controls.
func Charts(s *engine.Session) *dom.Node {
	// Bar chart with add/randomize controls
	barData := dom.NewBoxSignal(components.BarChartData{
		Labels: []string{"A", "B", "C", "D", "E"},
		Values: []float64{30, 50, 20, 60, 40},
		Title:  "Interactive Bar Chart",
	})

	// Line chart with auto-updating time series
	lineData := dom.NewBoxSignal(components.LineChartData{
		Labels: []string{"0s", "2s", "4s", "6s", "8s", "10s"},
		Series: []components.LineSeries{
			{Name: "CPU", Values: []float64{30, 45, 38, 52, 41, 48}, Color: "#3b82f6"},
			{Name: "Memory", Values: []float64{60, 62, 65, 63, 68, 70}, Color: "#ef4444"},
		},
		Title: "System Metrics (Live)",
	})

	// Pie chart with add/remove segments
	pieData := dom.NewBoxSignal(components.PieChartData{
		Labels: []string{"Chrome", "Firefox", "Safari", "Edge"},
		Values: []float64{65, 15, 12, 8},
		Title:  "Browser Market Share",
	})

	// Sparkline strip
	spark1 := dom.NewBoxSignal(components.SparkLineData{
		Values: []float64{10, 15, 12, 18, 14, 20, 16, 22, 19, 25},
		Color:  "#3b82f6",
	})
	spark2 := dom.NewBoxSignal(components.SparkLineData{
		Values: []float64{50, 48, 52, 45, 55, 42, 58, 40, 60, 38},
		Color:  "#ef4444",
	})
	spark3 := dom.NewBoxSignal(components.SparkLineData{
		Values: []float64{5, 8, 6, 9, 7, 10, 8, 11, 9, 12},
		Color:  "#22c55e",
	})

	// Auto-update line chart every 2s
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		tick := 6
		for {
			select {
			case <-s.Context().Done():
				return
			case <-ticker.C:
				tick++
				lineData.Update(func(d components.LineChartData) components.LineChartData {
					// Shift labels
					d.Labels = append(d.Labels[1:], fmt.Sprintf("%ds", tick*2))
					for i := range d.Series {
						base := d.Series[i].Values[len(d.Series[i].Values)-1]
						next := base + (rand.Float64()-0.5)*10
						if next < 0 {
							next = 5
						}
						if next > 100 {
							next = 95
						}
						d.Series[i].Values = append(d.Series[i].Values[1:], next)
					}
					return d
				})

				// Update sparklines
				spark1.Update(func(d components.SparkLineData) components.SparkLineData {
					d.Values = append(d.Values[1:], d.Values[len(d.Values)-1]+(rand.Float64()-0.4)*5)
					return d
				})
				spark2.Update(func(d components.SparkLineData) components.SparkLineData {
					d.Values = append(d.Values[1:], d.Values[len(d.Values)-1]+(rand.Float64()-0.5)*8)
					return d
				})
				spark3.Update(func(d components.SparkLineData) components.SparkLineData {
					d.Values = append(d.Values[1:], d.Values[len(d.Values)-1]+(rand.Float64()-0.4)*3)
					return d
				})
			}
		}
	}()

	return dom.Div(
		dom.Children(
			dom.H2(
				dom.Class(style.Classes(style.Text2Xl, style.FontBold, style.TextGray900, style.Mb6)),
				dom.Children(dom.Text("Chart Demos")),
			),

			// Bar chart + controls
			dom.Div(
				dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4, style.Mb6)),
				dom.Children(
					dom.Div(
						dom.Class(style.Classes(style.Flex, style.Gap2, style.Mb4)),
						dom.Children(
							dom.Button(
								dom.Class(style.Classes(style.BgGreen500, style.TextWhite, style.Px3, style.Py1, style.RoundedMd, style.TextSm)),
								dom.Children(dom.Text("Add Bar")),
								dom.OnClick(func(_ []byte) {
									barData.Update(func(d components.BarChartData) components.BarChartData {
										label := string(rune('A' + len(d.Labels)))
										d.Labels = append(d.Labels, label)
										d.Values = append(d.Values, float64(10+rand.IntN(60)))
										return d
									})
								}),
							),
							dom.Button(
								dom.Class(style.Classes(style.BgBlue500, style.TextWhite, style.Px3, style.Py1, style.RoundedMd, style.TextSm)),
								dom.Children(dom.Text("Randomize")),
								dom.OnClick(func(_ []byte) {
									barData.Update(func(d components.BarChartData) components.BarChartData {
										for i := range d.Values {
											d.Values[i] = float64(10 + rand.IntN(70))
										}
										return d
									})
								}),
							),
						),
					),
					components.BarChart(barData, 600, 300),
				),
			),

			// Line chart (auto-updating)
			dom.Div(
				dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4, style.Mb6)),
				dom.Children(
					components.LineChart(lineData, 600, 300),
				),
			),

			// Pie chart + controls
			dom.Div(
				dom.Class(style.Classes(style.Grid, style.GridCols2, style.Gap6, style.Mb6)),
				dom.Children(
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.Div(
								dom.Class(style.Classes(style.Flex, style.Gap2, style.Mb4)),
								dom.Children(
									dom.Button(
										dom.Class(style.Classes(style.BgGreen500, style.TextWhite, style.Px3, style.Py1, style.RoundedMd, style.TextSm)),
										dom.Children(dom.Text("Add Segment")),
										dom.OnClick(func(_ []byte) {
											pieData.Update(func(d components.PieChartData) components.PieChartData {
												d.Labels = append(d.Labels, fmt.Sprintf("New %d", len(d.Labels)))
												d.Values = append(d.Values, float64(5+rand.IntN(15)))
												return d
											})
										}),
									),
									dom.Button(
										dom.Class(style.Classes(style.BgRed500, style.TextWhite, style.Px3, style.Py1, style.RoundedMd, style.TextSm)),
										dom.Children(dom.Text("Remove Last")),
										dom.OnClick(func(_ []byte) {
											pieData.Update(func(d components.PieChartData) components.PieChartData {
												if len(d.Labels) > 1 {
													d.Labels = d.Labels[:len(d.Labels)-1]
													d.Values = d.Values[:len(d.Values)-1]
												}
												return d
											})
										}),
									),
								),
							),
							components.PieChart(pieData, 300, 300),
						),
					),

					// Sparklines strip
					dom.Div(
						dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
						dom.Children(
							dom.H3(
								dom.Class(style.Classes(style.FontBold, style.TextGray700, style.Mb4)),
								dom.Children(dom.Text("Sparklines (Live)")),
							),
							dom.Div(
								dom.Class(style.Classes(style.FlexCol, style.Flex, style.Gap4)),
								dom.Children(
									sparkRow("Requests/s", spark1),
									sparkRow("Latency (ms)", spark2),
									sparkRow("Errors/min", spark3),
								),
							),
						),
					),
				),
			),
		),
	)
}

func sparkRow(label string, data *dom.BoxSignal[components.SparkLineData]) *dom.Node {
	return dom.Div(
		dom.Class(style.Classes(style.Flex, style.ItemsCenter, style.Gap4)),
		dom.Children(
			dom.Span(
				dom.Class(style.Classes(style.TextSm, style.TextGray500)),
				dom.Style("width", "100px"),
				dom.Children(dom.Text(label)),
			),
			components.SparkLine(data, 200, 30),
		),
	)
}
