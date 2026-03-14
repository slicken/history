package charts

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/slicken/history"
)

// TradingView holds chart settings for LightweightCharts (TradingView)
type TradingView struct {
	SetWidth  string
	SetHeight string
	SetMargin string
	SMA       []int // Simple moving average periods (e.g. 20, 200)
	EMA       []int // Exponential moving average periods
}

// NewTradingView returns default chart settings
func NewTradingView() *TradingView {
	return &TradingView{
		SetWidth:  "90vw",
		SetHeight: "72vh",
		SetMargin: "25px",
		SMA:       []int{20, 200},
	}
}

// MakeOHLCTradingView formats bars for LightweightCharts (time in seconds)
// bars: index 0=newest, len-1=oldest. Output: oldest to newest (required by LightweightCharts)
func MakeOHLCTradingView(bars history.Bars) ([]byte, error) {
	count := int(math.Min(float64(len(bars)), float64(MAXLIMIT)))
	ohlc := make([]map[string]interface{}, count)
	for i := count - 1; i >= 0; i-- {
		bar := bars[i]
		ohlc[count-1-i] = map[string]interface{}{
			"time":  bar.Time.Unix(),
			"open":  bar.Open,
			"high":  bar.High,
			"low":   bar.Low,
			"close": bar.Close,
		}
	}
	return json.Marshal(ohlc)
}

// MakeSMATradingView computes SMA and formats for LightweightCharts line series
// bars: index 0=newest, len-1=oldest. Output: oldest to newest
func MakeSMATradingView(bars history.Bars, period int) ([]byte, error) {
	if period <= 0 || len(bars) < period {
		return []byte("[]"), nil
	}
	count := int(math.Min(float64(len(bars)), float64(MAXLIMIT)))
	sma := make([]map[string]interface{}, 0, count-period+1)
	for i := count - 1; i >= period-1; i-- {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += bars[j].Close
		}
		sma = append(sma, map[string]interface{}{
			"time":  bars[i].Time.Unix(),
			"value": sum / float64(period),
		})
	}
	return json.Marshal(sma)
}

// MakeEMATradingView computes EMA and formats for LightweightCharts line series
// bars: index 0=newest, len-1=oldest. Output: oldest to newest
func MakeEMATradingView(bars history.Bars, period int) ([]byte, error) {
	if period <= 0 || len(bars) < period {
		return []byte("[]"), nil
	}
	mult := 2.0 / float64(period+1)
	count := int(math.Min(float64(len(bars)), float64(MAXLIMIT)))
	ema := make([]map[string]interface{}, 0, count-period+1)
	sum := 0.0
	for j := count - period; j < count; j++ {
		sum += bars[j].Close
	}
	prevEMA := sum / float64(period)
	ema = append(ema, map[string]interface{}{"time": bars[count-1].Time.Unix(), "value": prevEMA})
	for i := count - 2; i >= period-1; i-- {
		prevEMA = (bars[i].Close-prevEMA)*mult + prevEMA
		ema = append(ema, map[string]interface{}{"time": bars[i].Time.Unix(), "value": prevEMA})
	}
	return json.Marshal(ema)
}

func (c *TradingView) smaScript(bars history.Bars) string {
	if len(c.SMA) == 0 && len(c.EMA) == 0 {
		return ""
	}
	var s string
	smaColors := []string{"#2196F3", "#f44336"} // blue, red
	ci := 0
	for _, period := range c.SMA {
		data, err := MakeSMATradingView(bars, period)
		if err != nil || len(data) <= 2 {
			continue
		}
		color := "#787b86"
		if ci < len(smaColors) {
			color = smaColors[ci]
		}
		ci++
		s += fmt.Sprintf("\n			var sma%d=chart.addLineSeries({color:'%s',lineWidth:1,lastValueVisible:false,priceLineVisible:false});sma%d.setData(%s);",
			period, color, period, string(data))
	}
	emaColors := []string{"#2196F3", "#f44336"}
	ci = 0
	for _, period := range c.EMA {
		data, err := MakeEMATradingView(bars, period)
		if err != nil || len(data) <= 2 {
			continue
		}
		color := "#787b86"
		if ci < len(emaColors) {
			color = emaColors[ci]
		}
		ci++
		s += fmt.Sprintf("\n			var ema%d=chart.addLineSeries({color:'%s',lineWidth:1,lastValueVisible:false,priceLineVisible:false});ema%d.setData(%s);",
			period, color, period, string(data))
	}
	return s
}

func (c *TradingView) markersScript(markersJS string, hasMarkers bool) string {
	if !hasMarkers {
		return ""
	}
	return "\n			var markers=[];" + markersJS + "if(markers.length>0)series.setMarkers(markers);"
}

func generateEventMarkers(events history.Events) string {
	var markers string
	for _, event := range events {
		color := "#FFFFFF"
		shape := "circle"
		position := "aboveBar"

		switch event.Type {
		case history.MARKET_BUY, history.LIMIT_BUY, history.STOP_BUY:
			shape = "arrowUp"
			position = "belowBar"
			color = "#26a69a"
		case history.MARKET_SELL, history.LIMIT_SELL, history.STOP_SELL:
			shape = "arrowDown"
			position = "aboveBar"
			color = "#ef5350"
		case history.CLOSE:
			shape = "circle"
			position = "aboveBar"
			color = "#2196F3"
		}

		markers += fmt.Sprintf(`markers.push({time:%d,position:'%s',color:'%s',shape:'%s'});`,
			event.Time.Unix(), position, color, shape) + "\n"
	}
	return markers
}

// MakeHeader creates chart headers
func (c *TradingView) MakeHeader() ([]byte, error) {
	return []byte(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width,initial-scale=1.0">
	<script src="https://unpkg.com/lightweight-charts@3.8.0/dist/lightweight-charts.standalone.production.js"></script>
	<style>
		body{margin:0;padding:0;font-family:-apple-system,BlinkMacSystemFont,'Trebuchet MS',Roboto,sans-serif;background:#0f0f0f;display:flex;flex-direction:column;align-items:center;}
		.chart-block{margin:` + c.SetMargin + `;}
		.chart-title{color:#787b86;font-size:14px;margin-bottom:8px;text-align:left;}
		.charts{margin:0;width:` + c.SetWidth + `;height:` + c.SetHeight + `;min-width:320px;min-height:280px;}
	</style>
</head>
<body>`), nil
}

// MakeChart creates a chart for a symbol
func (c *TradingView) MakeChart(name string, bars history.Bars, events history.Events) ([]byte, error) {
	if len(bars) == 0 {
		return nil, errors.New("no price data")
	}

	ohlc, err := MakeOHLCTradingView(bars)
	if err != nil {
		return nil, err
	}

	markersJS := generateEventMarkers(events)
	hasMarkers := len(events) > 0
	smaScript := c.smaScript(bars)

	return []byte(`
	<div class="chart-block">
		<div class="chart-title">` + name + `</div>
		<div class="charts" id="` + name + `"></div>
	</div>
	<script>
	(function(){
		var id='` + name + `';
		function init(retries){
			retries=retries||0;
			var el=document.getElementById(id);
			if(!el||el.offsetWidth<=0||el.offsetHeight<=0){
				if(retries<60)requestAnimationFrame(function(){init(retries+1);});
				return;
			}
			var container=el;
			var chart=LightweightCharts.createChart(container,{
				width:container.clientWidth,
				height:container.clientHeight,
				layout:{background:{color:"#0f0f0f"},textColor:"#787b86"},
				grid:{vertLines:{color:"#2B2B43"},horzLines:{color:"#2B2B43"}},
				crosshair:{mode:LightweightCharts.CrosshairMode.Normal},
				rightPriceScale:{borderColor:"#787b86",scaleMargins:{top:0.1,bottom:0.1}},
				timeScale:{borderColor:"#787b86",timeVisible:true,secondsVisible:false}
			});
			window.addEventListener('resize',function(){chart.applyOptions({width:container.clientWidth,height:container.clientHeight});});
			var series=chart.addCandlestickSeries({
				upColor:'#26a69a',downColor:'#ef5350',
				borderUpColor:'#26a69a',borderDownColor:'#ef5350',
				wickUpColor:'#26a69a',wickDownColor:'#ef5350',
				priceLineVisible:false
			});
			var ohlc=` + string(ohlc) + `;
			series.setData(ohlc);` + smaScript + c.markersScript(markersJS, hasMarkers) + `
		}
		if(document.readyState==='loading'){document.addEventListener('DOMContentLoaded',function(){requestAnimationFrame(init);});}
		else{requestAnimationFrame(init);}
	})();
	</script>`), nil
}

// BuildCharts builds multiple charts (implements ChartBuilder interface)
func (c *TradingView) BuildCharts(m map[string]history.Bars, events map[string]history.Events) ([]byte, error) {
	if len(m) == 0 {
		return []byte(`no charts history`), errors.New("no charts history")
	}

	buf, err := c.MakeHeader()
	if err != nil {
		return nil, err
	}

	if len(events) > 0 {
		for symbol, ev := range events {
			bars, ok := m[symbol]
			if !ok {
				continue
			}
			chart, err := c.MakeChart(symbol, bars, ev)
			if err != nil {
				continue
			}
			buf = append(buf, chart...)
		}
	} else {
		for symbol, bars := range m {
			chart, err := c.MakeChart(symbol, bars, nil)
			if err != nil {
				continue
			}
			buf = append(buf, chart...)
		}
	}

	buf = append(buf, []byte("\n</body>\n</html>")...)
	return buf, nil
}
