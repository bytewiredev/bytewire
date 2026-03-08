package pages

import (
	"math/rand/v2"
	"time"

	"github.com/bytewiredev/bytewire/pkg/components"
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/engine"
	"github.com/bytewiredev/bytewire/pkg/style"
)

// Home renders a dashboard page with stat cards, sparklines, and live-updating charts.
func Home(s *engine.Session) *dom.Node {
	// Stat cards with sparklines
	usersData := dom.NewBoxSignal(components.SparkLineData{
		Values: []float64{120, 132, 101, 134, 145, 160, 155},
		Color:  "#3b82f6",
	})
	revenueData := dom.NewBoxSignal(components.SparkLineData{
		Values: []float64{5200, 5400, 5100, 5800, 6200, 6100, 6500},
		Color:  "#22c55e",
	})
	ordersData := dom.NewBoxSignal(components.SparkLineData{
		Values: []float64{42, 38, 45, 50, 48, 52, 55},
		Color:  "#eab308",
	})
	convData := dom.NewBoxSignal(components.SparkLineData{
		Values: []float64{3.2, 3.5, 3.1, 3.8, 4.0, 3.9, 4.2},
		Color:  "#8b5cf6",
	})

	// Bar chart — weekly signups
	barData := dom.NewBoxSignal(components.BarChartData{
		Labels: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		Values: []float64{45, 62, 38, 71, 55, 33, 48},
		Title:  "Weekly Signups",
	})

	// Line chart — revenue (2 series)
	lineData := dom.NewBoxSignal(components.LineChartData{
		Labels: []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
		Series: []components.LineSeries{
			{Name: "Revenue", Values: []float64{4200, 5100, 4800, 6200, 5900, 7100}, Color: "#3b82f6"},
			{Name: "Expenses", Values: []float64{3100, 3400, 3200, 3800, 3600, 4000}, Color: "#ef4444"},
		},
		Title: "Revenue vs Expenses",
	})

	// Live update every 2 seconds.
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.Context().Done():
				return
			case <-ticker.C:
				// Append a random value to sparklines and shift window.
				usersData.Update(func(d components.SparkLineData) components.SparkLineData {
					d.Values = append(d.Values[1:], float64(100+rand.IntN(80)))
					return d
				})
				revenueData.Update(func(d components.SparkLineData) components.SparkLineData {
					d.Values = append(d.Values[1:], float64(4800+rand.IntN(2000)))
					return d
				})
				ordersData.Update(func(d components.SparkLineData) components.SparkLineData {
					d.Values = append(d.Values[1:], float64(35+rand.IntN(25)))
					return d
				})
				convData.Update(func(d components.SparkLineData) components.SparkLineData {
					d.Values = append(d.Values[1:], 2.5+rand.Float64()*2.5)
					return d
				})

				// Randomize bar chart.
				barData.Update(func(d components.BarChartData) components.BarChartData {
					for i := range d.Values {
						d.Values[i] = float64(20 + rand.IntN(60))
					}
					return d
				})
			}
		}
	}()

	return dom.Div(
		dom.Children(
			dom.H2(
				dom.Class(style.Classes(style.Text2Xl, style.FontBold, style.TextGray900, style.Mb6)),
				dom.Children(dom.Text("Dashboard")),
			),

			// Stat cards row
			dom.Div(
				dom.Class(style.Classes(style.Grid, style.GridCols4, style.Gap6, style.Mb6)),
				dom.Children(
					statCard("Active Users", "155", usersData),
					statCard("Revenue", "$6,500", revenueData),
					statCard("Orders", "55", ordersData),
					statCard("Conversion", "4.2%", convData),
				),
			),

			// Charts row
			dom.Div(
				dom.Class(style.Classes(style.Grid, style.GridCols2, style.Gap6)),
				dom.Children(
					chartCard(components.BarChart(barData, 500, 300)),
					chartCard(components.LineChart(lineData, 500, 300)),
				),
			),
		),
	)
}

func statCard(label, value string, sparkData *dom.BoxSignal[components.SparkLineData]) *dom.Node {
	return dom.Div(
		dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
		dom.Children(
			dom.P(
				dom.Class(style.Classes(style.TextSm, style.TextGray500)),
				dom.Children(dom.Text(label)),
			),
			dom.P(
				dom.Class(style.Classes(style.Text2Xl, style.FontBold, style.TextGray900)),
				dom.Children(dom.Text(value)),
			),
			components.SparkLine(sparkData, 120, 30),
		),
	)
}

func chartCard(chart *dom.Node) *dom.Node {
	return dom.Div(
		dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P4)),
		dom.Children(chart),
	)
}

