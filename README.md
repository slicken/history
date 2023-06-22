# HISTORY

manages and stores market history data.<br>
downloads new bars as there are new bars avaliable.<br>
create strategies, run backtests nad stream bars data<br>
plot to http with highcharts<br><br>


# run example app

use the examples
```
git clone https://slicken/history.git

cd history/examples

go build -o app

./app -quote=BTC -limit=200
```

view preformance by days, vist 127.0.0.1:8080/top/N = days<br>
![Alt text](examples/top30.png?raw=true "127.0.0.1/top/30")

- list all bars: 127.0.0.1:8080/
- normal backtest: 127.0.0.1:8080/test
- portfolio test: 127.0.0.1:8080/backtest
- top 30 days preformers: 127.0.0.1:8080/top/30

# example file
view examples/main.go for details and exmpanation.
