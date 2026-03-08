package components

// BarChartData holds data for a bar chart.
type BarChartData struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
	Colors []string  `json:"colors,omitempty"`
	Title  string    `json:"title,omitempty"`
}

// LineChartData holds data for a line chart with multiple series.
type LineChartData struct {
	Labels []string     `json:"labels"`
	Series []LineSeries `json:"series"`
	Title  string       `json:"title,omitempty"`
}

// LineSeries is a single data series within a line chart.
type LineSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color"`
}

// PieChartData holds data for a pie or donut chart.
type PieChartData struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
	Colors []string  `json:"colors,omitempty"`
	Donut  bool      `json:"donut,omitempty"`
	Title  string    `json:"title,omitempty"`
}

// SparkLineData holds data for a minimal inline sparkline.
type SparkLineData struct {
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"`
}
