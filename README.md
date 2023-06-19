# HISTORY

manages and saves market history data.<br>
create single/multi strategies on history(bars) or whole data stuct.<br>
run backtest or stream the data<br>
plot to http<br><br>


# getting started

use the examples
```
git clone https://slicken/history.git

cd history/examples

go build -o app

./app -quote=BTC -limit=300
```
![main backtest db](docs/images/backtestDB.png)

# view
creeate variables for this example
```
var (
	hist          = new(history.History)       // main struct to handle all bars data
	eventListener = new(history.EventListener) // eventlistener is looking for new bars and runs strategies on them
	events        = history.Events             // store our events here, if we want to save them
	strategy      = &Engulfing{}               // engulfing strategy (create you own) strategies 
	chart         = highcharts.DefaultChart()  // use highcharts for plotting
)
```
# getting started
```
func main() 
	----------------------------------------------------------------------------------------------
	// add your downloader intefcace so we can update bars.
	// ----------------------------------------------------------------------------------------------
	hist.Downloader = &Binance{}
	// ----------------------------------------------------------------------------------------------
	// change the default directory to store history data from exchange
	// ----------------------------------------------------------------------------------------------
	hist.SetDataDir("download")
	// ----------------------------------------------------------------------------------------------
	// update bars when there is a fresh bar for your symbol
	// ----------------------------------------------------------------------------------------------
	hist.Update(true)
	// ----------------------------------------------------------------------------------------------
	// load symbols. use a list to load multiple.
	// ----------------------------------------------------------------------------------------------
	hist.Load("BTCUSD1d", "ETHUSD1d", "RNDRUSD1d", "APTUSD1d")
	// ----------------------------------------------------------------------------------------------
	// limit bars
	// ----------------------------------------------------------------------------------------------
	hist.Limit(300)
	// ----------------------------------------------------------------------------------------------
	// add strategy to event listener
	// ----------------------------------------------------------------------------------------------
	eventListener.Add(strategy)
	// ----------------------------------------------------------------------------------------------
	// start event listener or all bars that 'hist' have loaded
	// ----------------------------------------------------------------------------------------------
	eventListener.Start(hist, events)
	// ----------------------------------------------------------------------------------------------
	// highchart settings (highcharts)
	// ----------------------------------------------------------------------------------------------
	chart.Type = highcharts.ChartType(conf.ctype)
	chart.SMA = []int{20, 200}
	// ----------------------------------------------------------------------------------------------
	// http routes for visual results and backtesting
	// ----------------------------------------------------------------------------------------------
	http.HandleFunc("/", httpIndex)
	http.HandleFunc("/test", httpTest)
	http.HandleFunc("/backtest", httpBacktest)
	http.HandleFunc("/top/", httpTopPreformers) // top preformers for x days
	http.HandleFunc("/gainers/", httpTopGainers)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## plot results ##
vist https://localhost:8080/top/90<br><br>

shows best preformers for last 90 days<br>

<import image="">...</umage>
