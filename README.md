# HISTORY - !NOT FINISHED!

manages and stores market history bars data<br>
downloads and updates when there is new bars avaliable<br>
create strategies and other events, run backtests and stream data<br>
plot to http with highcharts<br><br>


# run example app

use the examples

```
git clone https://slicken/history.git
cd history/examples
go build -o app
./app -quote=BTC -limit=200
```

- list all history bars: 127.0.0.1:8080/
- run normal backtest: 127.0.0.1:8080/test
- portfolio test: 127.0.0.1:8080/backtest
- list top preformers (N days): 127.0.0.1:8080/top/30

![Alt text](examples/top30.png?raw=true "127.0.0.1/top/30")

# example file
view examples/main.go for more details and examples.
