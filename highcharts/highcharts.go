package highcharts

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"

	"github.com/slicken/history"
)

// MAXLIMIT on chart data arrays
const MAXLIMIT = 10000

// HighChart holds chart settings
type HighChart struct {
	// Type sets chart type  (Candlestick|Ohlc|Line|Spline)
	Type ChartType
	SMA  []int // Simple moving averages
	EMA  []int // Exponential moving averages
	// Volume axis
	Volume bool
	// Volume SMA
	VolumeSMA int
	// Shadows styles chart
	Shadow bool
	// Chart HTTP settings
	SetWidth, SetHeight, SetMargin string
}

// ChartType ..
type ChartType string

const (
	// Candlestick type
	Candlestick ChartType = "candlestick"
	// Ohlc type
	Ohlc ChartType = "ohlc"
	// Line type
	Line ChartType = "line"
	// Spline smooth line type
	Spline ChartType = "spline"
)

// DefaultChart returns default chart settings
func DefaultChart() *HighChart {
	return &HighChart{
		Type:      Candlestick,
		Volume:    true,
		Shadow:    false,
		SetWidth:  "56%",
		SetHeight: "72%",
		SetMargin: "25px",
	}
}

/*
	highcharts series

var ohlc = `[
{"x":1547337600000,"open":3584.1,"high":3611.1,"low":3441.3,"close":3476.81,dataLabels: { enabled: true }},
{"x":1547683200000,"open":3591.84,"high":3634.7,"low":3530.39,"close":3616.21,"name":"test","color":"black"},
*/

// MakeOHLC = price
func MakeOHLC(bars history.Bars) ([]byte, error) {
	var data []interface{}

	count := int(math.Min(float64(len(bars)), MAXLIMIT))
	for i := count - 1; i >= 0; i-- {
		v := []interface{}{bars[i].Time.Unix() * 1000, bars[i].Open, bars[i].High, bars[i].Low, bars[i].Close}
		data = append(data, v)
	}
	return json.Marshal(&data)
}

// MakeVolume ..
func MakeVolume(bars history.Bars) ([]byte, error) {
	var vol []interface{}

	count := int(math.Min(float64(len(bars)), MAXLIMIT))
	for i := count - 1; i >= 0; i-- {
		v := []interface{}{bars[i].Time.Unix() * 1000, bars[i].Volume}
		vol = append(vol, v)
	}
	return json.Marshal(&vol)
}

// MakeEventFlags events
func MakeEventFlags(events history.Events) ([]string, []string) {
	var buy, sell = make([]string, 0), make([]string, 0)

	for _, event := range events {
		// s := fmt.Sprintf(`{"x":%d,"title":%q,"text":%q},`, event.Time.Unix()*1000, EventTypes[event.Type], fmt.Sprintf("%s\n%s", event.Title, event.Text))

		if event.Type == 0 || event.Type == 2 || event.Type == 5 {
			s := fmt.Sprintf(`{"x":%d,"title":"B","text":%q},`, event.Time.Unix()*1000, (event.Name + " " + history.EventTypes[event.Type] + " " + event.Text))
			buy = append(buy, s)
		}
		if event.Type == 1 || event.Type == 3 || event.Type == 4 {
			s := fmt.Sprintf(`{"x":%d,"title":"S","text":%q},`, event.Time.Unix()*1000, (event.Name + " " + history.EventTypes[event.Type] + " " + event.Text))
			sell = append(sell, s)
		}
	}

	return buy, sell
}

// MakeHeader creates chart headers
func (c *HighChart) MakeHeader() ([]byte, error) {
	// <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
	return []byte(`
	<head>
		<meta name="viewport" content="width=device-width"/>
		<script src="https://code.highcharts.com/stock/highstock.js"></script>
		<script src="https://code.highcharts.com/modules/data.js"></script>
		<script src="https://code.highcharts.com/stock/indicators/indicators.js"></script>
		<script src="https://code.highcharts.com/stock/indicators/ema.js"></script>
	</head>
	<style>
		html{font-family: 'Lato',sans-serif;}
		body{
			overflow: auto;
			background: whitesmoke;
	
			display: flex;
			flex-direction: column;
			align-items: center;
		}
		.charts {
			width: ` + c.SetWidth + `;
			height: ` + c.SetHeight + `;
			margin: ` + c.SetMargin + `;
		}
	 </style>`), nil
}

// MakeChart template
func (c *HighChart) MakeChart(name string, bars history.Bars, events history.Events) ([]byte, error) {
	if name == "" {
		name = "unknown"
	}

	ohlc, err := MakeOHLC(bars)
	if err != nil {
		return nil, err
	}
	if len(ohlc) == 0 {
		return nil, errors.New("no price data")
	}

	return []byte(`
	<div class="charts" id="` + name + `"></div>
	<script>

	Highcharts.setOptions({
		lang: {
			rangeSelectorZoom: ''
		}
	});

	Highcharts.stockChart('` + name + `', {
		credits: false,

		title: {
			text: '` + name + `',
			align: 'left',
			floating: true,
			style: {
			  	color: '#707070',
			  	fontSize: '12px',
			  	fontWeight: 'normal',
			  	fontStyle: 'none',
			  	textTransform: 'none',
			}
		},
		chart: {
			borderWidth: 0,
			alignTicks: false,
			spacing: 15,
			zoomType: 'x',
		},` +

		// volume axis if enabled
		func() string {
			if c.Volume {
				return `
				yAxis: [{
					gridLineWidth: 0,
					lineWidth: 0,
					height: '70%',
				}, {
					gridLineWidth: 0,
					lineWidth: 0,
					height: '30%',
					top: '70%',
				}],`
			}
			return `
			yAxis: {
				gridLineWidth: 0,
				lineWidth: 0,
			},`
		}() + `

		tooltip: {
			backgroundColor: 'white',
			borderWidth: 0,
			crosshairs: [false, false], // vertial, horizontal
			hideDelay: 0,
			snap: 0,
			shared: true, // share charts
		},

		rangeSelector: {
			enabled: false, // enable?
			inputEnabled: false,
            selected: 3,
		},

		navigator: {
			enabled: false,
			adaptToUpdatedData: true,
			series: {
				type: 'spline',
			},
			xAxis: {
			  	gridLineWidth: 0,
			},
		},

		scrollbar: {
			height: 0,
			buttonArrowColor: 'transparent',
			liveRedraw: true,
		},

		plotOptions: {
			sma: {
				enableMouseTracking: false,
				lineWidth: 1,
			},
			ema: {
				enableMouseTracking: false,
				lineWidth: 1,
			},
			series: {
				turboThreshold: 0,
				dataGrouping: {
					enabled: false,
				},
				marker: {
					enabled: false,
				},
			},
		},

		series: [{
            type: '` + string(c.Type) + `',
			name: '` + name + `',
			id: '` + name + `',
			zIndex: 5,
			data: ` + string(ohlc) + `,
			shadow: ` + fmt.Sprintf("%v", c.Shadow) + `,` +

		func() (s string) {
			// flags data
			flagB, flagS := MakeEventFlags(events)

			// B flag
			if len(flagB) > 0 {
				s += `
				}, {
					type: 'flags',
					data: ` + fmt.Sprintf("%s", flagB) + `,
					zIndex: 19,
					onSeries: '` + name + `',
					shape: 'circlepin',
					color: 'green',
					fillColor: 'green',
					style: {
						color: 'white'
					},`
			}
			// S flag
			if len(flagS) > 0 {
				s += `
				}, {
					type: 'flags',
					data: ` + fmt.Sprintf("%s", flagS) + `,
					zIndex: 20,
					onSeries: '` + name + `',
					shape: 'circlepin',
					color: '#f45b5b',
					fillColor: '#f45b5b',
					style: {
						color: 'white'
					},`
			}

			// volume
			if c.Volume {
				// calc volume data
				volume, _ := MakeVolume(bars)
				s += `
				}, {
					type: 'column',
					name: 'Volume',
					id: 'vol',
					data: ` + string(volume) + `,
					yAxis: 1,
					zIndex: 1,
					shadow: ` + fmt.Sprintf("%v", c.Shadow) + `,`
				// volume sma
				if c.VolumeSMA > 0 {
					s += `
					}, {
						type: 'sma',
						linkedTo: 'vol',
						params: {
							period: ` + fmt.Sprintf("%v", c.VolumeSMA) + `,
						},
						yAxis: 1,
						zIndex: 2,`
				}
			}
			// sma's
			if len(c.SMA) > 0 {
				for _, v := range c.SMA {
					s += `
					}, {
						type: 'sma',
						linkedTo: '` + name + `',
						params: {
							period: ` + fmt.Sprintf("%v", v) + `,
						},
						zIndex: 4,`
				}
			}
			// ema's
			if len(c.EMA) > 0 {
				for _, v := range c.EMA {
					s += `
					}, {
						type: 'ema',
						linkedTo: '` + name + `',
						params: {
							period: ` + fmt.Sprintf("%v", v) + `,
						},
						zIndex: 3,`
				}
			}
			return
		}() +

		`}]
	});
	</script>`), nil
}

/*
	plotOptions: {
		series: {
			dataLabels: {
				useHTML: true,
				shape: 'callout',
				padding: 7,
				borderWidth: 1,
				borderRadius: 2,
				borderColor: 'black',
				backgroundColor: 'white',

			},
		},
	},
*/

// BuildCharts builds multiple charts
func (c *HighChart) BuildCharts(m map[string]history.Bars, events map[string]history.Events) (buf []byte, err error) {
	if len(m) == 0 {
		return []byte(`no charts history`), errors.New("no charts history")
	}

	buf, err = c.MakeHeader()
	if err != nil {
		return nil, err
	}

	if len(events) > 0 {
		// make map of all events for given symbol
		for symbol, ev := range events {
			// get bars of symbol
			bars, ok := m[symbol]
			if !ok {
				continue
			}
			// make chart with events included
			chart, err := c.MakeChart(symbol, bars, ev)
			if err != nil {
				log.Println(err)
			}
			// append to slice
			buf = append(buf, chart...)
		}

	} else {
		for symbol, bars := range m {

			chart, err := c.MakeChart(symbol, bars, nil)
			if err != nil {
				log.Println(err)
			}
			// append to slice
			buf = append(buf, chart...)
		}
	}

	return buf, err
}
