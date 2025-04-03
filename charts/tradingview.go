package charts

// package tradingview

// import (
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"log"
// 	"math"

// 	"github.com/slicken/history"
// )

// const MAXLENGTH = 10000

// type Chart struct {
// 	SetWidth, SetHeight, SetMargin string
// }

// // DefaultChart returns default chart settings
// func DefaultChart() *Chart {
// 	return &Chart{
// 		SetWidth:  "100%",
// 		SetHeight: "100%",
// 		SetMargin: "0",
// 	}
// }

// // MakeHeader creates chart headers
// func (c *Chart) MakeHeader(title string) ([]byte, error) {
// 	return []byte(`
// <!DOCTYPE html>
// <html>
// <head>
// 	<meta charset="utf-8">
// 	<meta name="viewport" content="width=device-width,initial-scale=1.0,maximum-scale=1.0,minimum-scale=1.0">
// 	<title>` + title + `</title>
// 	<script src="https://unpkg.com/lightweight-charts@3.8.0/dist/lightweight-charts.standalone.production.js"></script>
// 	<style>
// 		body { margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Trebuchet MS', Roboto, Ubuntu, sans-serif; }
// 		.charts { margin: ` + c.SetMargin + `; width: 100vw; height: 100vh; }
// 	</style>
// </head>
// `), nil
// }

// func (c *Chart) MakeChart(name string, bars []byte, events history.Events) ([]byte, error) {
// 	return []byte(`
// <body>
// <div class="charts" id="` + name + `"></div>
// <script type="text/javascript">
// 	const container = document.getElementById('` + name + `');
// 	const chart = LightweightCharts.createChart(container, {
// 	width: container.clientWidth,
// 	height: container.clientHeight,
// 	layout: {
// 		background: { color: "#0f0f0f" },
// 		textColor: "#787b86",
// 	},
// 	grid: {
// 		vertLines: { color: "transparent" },
// 		horzLines: { color: "transparent" },
// 	},
// 	crosshair: {
// 		mode: LightweightCharts.CrosshairMode.Normal,
// 	},
// 	rightPriceScale: {
// 		borderColor: "#787b86",
// 	},
// 	timeScale: {
// 		borderColor: "#787b86",
// 		rightOffset: 10,
// 	},
//   	rightPriceScale: {
//     	scaleMargins: {
//       		top: 0.1,
//       		bottom: 0.1,
//     	},
//   	},
// });

// window.addEventListener('resize', () => {
// 	chart.applyOptions({
// 		width: container.clientWidth,
// 		height: container.clientHeight,
// 	});
// });

// var series = chart.addCandlestickSeries({
// 	upColor: '#00994c',
// 	downColor: '#a50d0d',
// 	borderUpColor: '#00994c',
// 	borderDownColor: '#a50d0d',
// 	wickUpColor: '#00994c',
// 	wickDownColor: '#a50d0d',
// });

// var ohlc = ` + string(bars) + `;
// series.setData(ohlc);

// // Add markers for events
// const markers = [];
// ` + generateEventMarkers(events) + `
// if (markers.length > 0) {
// 	series.setMarkers(markers);
// }

// </script>
// </body>
// </html>
// `), nil
// }

// func generateEventMarkers(events history.Events) string {
// 	var markers string
// 	for _, event := range events {
// 		color := "#FFFFFF" // default white
// 		shape := "circle"
// 		position := "aboveBar"

// 		switch event.Type {
// 		case history.MARKET_BUY:
// 			shape = "arrowUp"
// 			position = "belowBar"
// 		case history.MARKET_SELL:
// 			shape = "arrowDown"
// 			position = "aboveBar"
// 		case history.LIMIT_BUY:
// 			shape = "arrowUp"
// 			position = "belowBar"
// 		case history.LIMIT_SELL:
// 			shape = "arrowDown"
// 			position = "aboveBar"
// 		case history.CLOSE_BUY:
// 			shape = "x"
// 			position = "aboveBar"
// 		case history.CLOSE_SELL:
// 			shape = "x"
// 			position = "belowBar"
// 		}

// 		// Format time as Unix timestamp in seconds
// 		marker := fmt.Sprintf(`markers.push({
// 			time: %d,
// 			position: '%s',
// 			color: '%s',
// 			shape: '%s',
// 			size: 1
// 		});`, event.Time.Unix(), position, color, shape)

// 		markers += marker + "\n"
// 	}
// 	return markers
// }

// // BuildChart creates a complete chart with data and events
// func (c *Chart) BuildChart(title string, bars history.Bars, events history.Events) (buf []byte, err error) {
// 	if len(bars) == 0 {
// 		return nil, errors.New("no chart data")
// 	}

// 	// Use the data directly from history.Bars
// 	length := int(math.Min(float64(len(bars)), MAXLENGTH))
// 	ohlcData := make([]map[string]interface{}, length)
// 	for i := 0; i < length; i++ {
// 		bar := bars[len(bars)-1-i]
// 		ohlcData[i] = map[string]interface{}{
// 			"time":  bar.Time.Unix(),
// 			"open":  bar.Open,
// 			"high":  bar.High,
// 			"low":   bar.Low,
// 			"close": bar.Close,
// 		}
// 	}

// 	// Get the first and last bar times to ensure markers are within this range
// 	var firstTime, lastTime int64
// 	if len(ohlcData) > 0 {
// 		firstTime = ohlcData[0]["time"].(int64)
// 		lastTime = ohlcData[len(ohlcData)-1]["time"].(int64)
// 	}

// 	// Filter events to only include those within the bar time range
// 	filteredEvents := make(history.Events, 0)
// 	for _, event := range events {
// 		eventTime := event.Time.Unix()
// 		if eventTime >= firstTime && eventTime <= lastTime {
// 			filteredEvents = append(filteredEvents, event)
// 		}
// 	}

// 	data, err := json.Marshal(ohlcData)
// 	if err != nil {
// 		return nil, err
// 	}

// 	buf, err = c.MakeHeader(title)
// 	if err != nil {
// 		return nil, err
// 	}

// 	chart, err := c.MakeChart(title, data, filteredEvents)
// 	if err != nil {
// 		log.Println(err)
// 	}

// 	buf = append(buf, chart...)

// 	return buf, err
// }
