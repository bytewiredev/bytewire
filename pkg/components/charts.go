package components

import (
	"fmt"

	"github.com/bytewiredev/bytewire/pkg/dom"
)

// BarChart creates a canvas-based bar chart bound to a reactive data signal.
// The server sets data-bw-chart and data-bw-chart-data attributes; the WASM
// client renders using the Canvas 2D API.
func BarChart(data *dom.BoxSignal[BarChartData], width, height int) *dom.Node {
	return dom.Canvas(
		dom.Attr("width", fmt.Sprintf("%d", width)),
		dom.Attr("height", fmt.Sprintf("%d", height)),
		dom.Attr("data-bw-chart", "bar"),
		dom.AttrJSON[BarChartData](data, "data-bw-chart-data"),
	)
}

// LineChart creates a canvas-based line chart bound to a reactive data signal.
func LineChart(data *dom.BoxSignal[LineChartData], width, height int) *dom.Node {
	return dom.Canvas(
		dom.Attr("width", fmt.Sprintf("%d", width)),
		dom.Attr("height", fmt.Sprintf("%d", height)),
		dom.Attr("data-bw-chart", "line"),
		dom.AttrJSON[LineChartData](data, "data-bw-chart-data"),
	)
}

// PieChart creates a canvas-based pie/donut chart bound to a reactive data signal.
func PieChart(data *dom.BoxSignal[PieChartData], width, height int) *dom.Node {
	return dom.Canvas(
		dom.Attr("width", fmt.Sprintf("%d", width)),
		dom.Attr("height", fmt.Sprintf("%d", height)),
		dom.Attr("data-bw-chart", "pie"),
		dom.AttrJSON[PieChartData](data, "data-bw-chart-data"),
	)
}

// SparkLine creates a minimal canvas sparkline bound to a reactive data signal.
func SparkLine(data *dom.BoxSignal[SparkLineData], width, height int) *dom.Node {
	return dom.Canvas(
		dom.Attr("width", fmt.Sprintf("%d", width)),
		dom.Attr("height", fmt.Sprintf("%d", height)),
		dom.Attr("data-bw-chart", "sparkline"),
		dom.AttrJSON[SparkLineData](data, "data-bw-chart-data"),
	)
}
