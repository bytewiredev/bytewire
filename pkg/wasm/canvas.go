//go:build js && wasm

package wasm

import (
	"encoding/json"
	"math"
	"syscall/js"
)

// defaultColors is a palette used when chart data doesn't specify colors.
var defaultColors = []string{
	"#3b82f6", "#ef4444", "#22c55e", "#eab308", "#8b5cf6",
	"#ec4899", "#06b6d4", "#f97316", "#14b8a6", "#6366f1",
}

func getColor(colors []string, i int) string {
	if i < len(colors) && colors[i] != "" {
		return colors[i]
	}
	return defaultColors[i%len(defaultColors)]
}

// renderChart dispatches to the appropriate chart renderer based on the
// data-bw-chart attribute value.
func renderChart(el js.Value) {
	chartType := el.Call("getAttribute", "data-bw-chart")
	if chartType.IsNull() || chartType.IsUndefined() {
		return
	}
	dataAttr := el.Call("getAttribute", "data-bw-chart-data")
	if dataAttr.IsNull() || dataAttr.IsUndefined() {
		return
	}

	// Handle devicePixelRatio for retina displays.
	dpr := js.Global().Get("devicePixelRatio").Float()
	if dpr == 0 {
		dpr = 1
	}

	w := el.Get("width").Int()
	h := el.Get("height").Int()

	// Scale canvas for retina.
	el.Set("width", int(float64(w)*dpr))
	el.Set("height", int(float64(h)*dpr))
	el.Get("style").Set("width", js.ValueOf(w).Call("toString").String()+"px")
	el.Get("style").Set("height", js.ValueOf(h).Call("toString").String()+"px")

	ctx := el.Call("getContext", "2d")
	ctx.Call("scale", dpr, dpr)

	dataJSON := dataAttr.String()

	switch chartType.String() {
	case "bar":
		var data barChartData
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			return
		}
		renderBarChart(ctx, data, w, h)
	case "line":
		var data lineChartData
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			return
		}
		renderLineChart(ctx, data, w, h)
	case "pie":
		var data pieChartData
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			return
		}
		renderPieChart(ctx, data, w, h)
	case "sparkline":
		var data sparkLineData
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			return
		}
		renderSparkLine(ctx, data, w, h)
	}
}

func renderBarChart(ctx js.Value, data barChartData, w, h int) {
	if len(data.Values) == 0 {
		return
	}

	// Clear
	ctx.Call("clearRect", 0, 0, w, h)

	padding := 40
	chartW := w - padding*2
	chartH := h - padding*2
	barW := float64(chartW) / float64(len(data.Values))
	gap := barW * 0.2

	// Find max value.
	maxVal := 0.0
	for _, v := range data.Values {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	// Title
	if data.Title != "" {
		ctx.Set("fillStyle", "#111827")
		ctx.Set("font", "bold 14px system-ui, sans-serif")
		ctx.Set("textAlign", "center")
		ctx.Call("fillText", data.Title, w/2, 20)
	}

	// Bars
	for i, v := range data.Values {
		barH := (v / maxVal) * float64(chartH)
		x := float64(padding) + float64(i)*barW + gap/2
		y := float64(padding+chartH) - barH

		ctx.Set("fillStyle", getColor(data.Colors, i))
		ctx.Call("fillRect", x, y, barW-gap, barH)

		// Label
		if i < len(data.Labels) {
			ctx.Set("fillStyle", "#6b7280")
			ctx.Set("font", "11px system-ui, sans-serif")
			ctx.Set("textAlign", "center")
			ctx.Call("fillText", data.Labels[i], x+(barW-gap)/2, float64(h-8))
		}
	}

	// Y-axis line
	ctx.Call("beginPath")
	ctx.Set("strokeStyle", "#d1d5db")
	ctx.Call("moveTo", padding, padding)
	ctx.Call("lineTo", padding, padding+chartH)
	ctx.Call("stroke")
}

func renderLineChart(ctx js.Value, data lineChartData, w, h int) {
	if len(data.Series) == 0 || len(data.Labels) == 0 {
		return
	}

	ctx.Call("clearRect", 0, 0, w, h)

	padding := 40
	chartW := w - padding*2
	chartH := h - padding*2

	// Find global max.
	maxVal := 0.0
	for _, s := range data.Series {
		for _, v := range s.Values {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	// Title
	if data.Title != "" {
		ctx.Set("fillStyle", "#111827")
		ctx.Set("font", "bold 14px system-ui, sans-serif")
		ctx.Set("textAlign", "center")
		ctx.Call("fillText", data.Title, w/2, 20)
	}

	// Grid lines
	ctx.Set("strokeStyle", "#e5e7eb")
	ctx.Set("lineWidth", 1)
	for i := range 5 {
		y := float64(padding) + float64(i)*float64(chartH)/4.0
		ctx.Call("beginPath")
		ctx.Call("moveTo", padding, y)
		ctx.Call("lineTo", padding+chartW, y)
		ctx.Call("stroke")
	}

	// Draw each series.
	stepX := float64(chartW) / float64(len(data.Labels)-1)
	for si, series := range data.Series {
		color := series.Color
		if color == "" {
			color = getColor(nil, si)
		}
		ctx.Set("strokeStyle", color)
		ctx.Set("lineWidth", 2)
		ctx.Call("beginPath")

		for i, v := range series.Values {
			if i >= len(data.Labels) {
				break
			}
			x := float64(padding) + float64(i)*stepX
			y := float64(padding+chartH) - (v/maxVal)*float64(chartH)
			if i == 0 {
				ctx.Call("moveTo", x, y)
			} else {
				ctx.Call("lineTo", x, y)
			}
		}
		ctx.Call("stroke")

		// Points
		ctx.Set("fillStyle", color)
		for i, v := range series.Values {
			if i >= len(data.Labels) {
				break
			}
			x := float64(padding) + float64(i)*stepX
			y := float64(padding+chartH) - (v/maxVal)*float64(chartH)
			ctx.Call("beginPath")
			ctx.Call("arc", x, y, 3, 0, 2*math.Pi)
			ctx.Call("fill")
		}
	}

	// X-axis labels
	ctx.Set("fillStyle", "#6b7280")
	ctx.Set("font", "11px system-ui, sans-serif")
	ctx.Set("textAlign", "center")
	for i, label := range data.Labels {
		x := float64(padding) + float64(i)*stepX
		ctx.Call("fillText", label, x, float64(h-8))
	}

	// Legend
	if len(data.Series) > 1 {
		legendX := float64(padding)
		for si, series := range data.Series {
			color := series.Color
			if color == "" {
				color = getColor(nil, si)
			}
			ctx.Set("fillStyle", color)
			ctx.Call("fillRect", legendX, float64(h-25), 12, 12)
			ctx.Set("fillStyle", "#374151")
			ctx.Set("font", "11px system-ui, sans-serif")
			ctx.Set("textAlign", "left")
			ctx.Call("fillText", series.Name, legendX+16, float64(h-15))
			legendX += float64(len(series.Name)*7 + 30)
		}
	}
}

func renderPieChart(ctx js.Value, data pieChartData, w, h int) {
	if len(data.Values) == 0 {
		return
	}

	ctx.Call("clearRect", 0, 0, w, h)

	// Title
	titleOffset := 0
	if data.Title != "" {
		titleOffset = 25
		ctx.Set("fillStyle", "#111827")
		ctx.Set("font", "bold 14px system-ui, sans-serif")
		ctx.Set("textAlign", "center")
		ctx.Call("fillText", data.Title, w/2, 20)
	}

	cx := float64(w) / 2
	cy := float64(h+titleOffset) / 2
	radius := math.Min(float64(w), float64(h-titleOffset)) / 2 * 0.75

	// Sum values.
	total := 0.0
	for _, v := range data.Values {
		total += v
	}
	if total == 0 {
		return
	}

	startAngle := -math.Pi / 2
	for i, v := range data.Values {
		sliceAngle := (v / total) * 2 * math.Pi
		endAngle := startAngle + sliceAngle

		ctx.Set("fillStyle", getColor(data.Colors, i))
		ctx.Call("beginPath")
		ctx.Call("moveTo", cx, cy)
		ctx.Call("arc", cx, cy, radius, startAngle, endAngle)
		ctx.Call("closePath")
		ctx.Call("fill")

		// Label at midpoint of arc.
		if i < len(data.Labels) {
			midAngle := startAngle + sliceAngle/2
			labelR := radius * 0.65
			lx := cx + math.Cos(midAngle)*labelR
			ly := cy + math.Sin(midAngle)*labelR
			ctx.Set("fillStyle", "#ffffff")
			ctx.Set("font", "bold 11px system-ui, sans-serif")
			ctx.Set("textAlign", "center")
			ctx.Set("textBaseline", "middle")
			ctx.Call("fillText", data.Labels[i], lx, ly)
		}

		startAngle = endAngle
	}

	// Donut hole.
	if data.Donut {
		ctx.Set("fillStyle", "#ffffff")
		ctx.Call("beginPath")
		ctx.Call("arc", cx, cy, radius*0.5, 0, 2*math.Pi)
		ctx.Call("fill")
	}
}

func renderSparkLine(ctx js.Value, data sparkLineData, w, h int) {
	if len(data.Values) == 0 {
		return
	}

	ctx.Call("clearRect", 0, 0, w, h)

	color := data.Color
	if color == "" {
		color = "#3b82f6"
	}

	// Find min/max.
	minVal, maxVal := data.Values[0], data.Values[0]
	for _, v := range data.Values[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	valRange := maxVal - minVal
	if valRange == 0 {
		valRange = 1
	}

	padY := 2.0
	chartH := float64(h) - padY*2
	stepX := float64(w) / float64(len(data.Values)-1)

	ctx.Set("strokeStyle", color)
	ctx.Set("lineWidth", 1.5)
	ctx.Call("beginPath")

	for i, v := range data.Values {
		x := float64(i) * stepX
		y := padY + chartH - ((v-minVal)/valRange)*chartH
		if i == 0 {
			ctx.Call("moveTo", x, y)
		} else {
			ctx.Call("lineTo", x, y)
		}
	}
	ctx.Call("stroke")
}
