//go:build js && wasm

package wasm

// Chart data structs duplicated for WASM build to avoid importing
// pkg/components which has server-only dependencies.

type barChartData struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
	Colors []string  `json:"colors,omitempty"`
	Title  string    `json:"title,omitempty"`
}

type lineChartData struct {
	Labels []string       `json:"labels"`
	Series []lineSeriesWS `json:"series"`
	Title  string         `json:"title,omitempty"`
}

type lineSeriesWS struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color"`
}

type pieChartData struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
	Colors []string  `json:"colors,omitempty"`
	Donut  bool      `json:"donut,omitempty"`
	Title  string    `json:"title,omitempty"`
}

type sparkLineData struct {
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"`
}
