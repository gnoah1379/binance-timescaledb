package main

import (
	"log/slog"
	"os"
	"os/signal"
	"time"
)

func main() {
	db := OpenDB(Config{
		DBName: "gnoah_trading_system",
		DBUser: "postgres",
		DBPwd:  "hoang123",
		DBHost: "127.0.0.1",
		DBPort: 5432,
	})
	repo := Repository{db: db}
	binance := NewSpotBinance()

	err := binance.ConnectStream()
	if err != nil {
		panic(err)
	}
	symbols, err := repo.GetEnabledSymbols()
	if err != nil {
		panic(err)
	}

	binance.candlestickCallback = func(data WsKline) {
		if !data.IsFinal {
			return
		}
		stick := Candlestick{
			Time:   time.UnixMilli(data.StartTime),
			Symbol: data.Symbol,
			Open:   SafeParseFloat(data.Open),
			High:   SafeParseFloat(data.High),
			Low:    SafeParseFloat(data.Low),
			Close:  SafeParseFloat(data.Close),
			Volume: SafeParseFloat(data.Volume),
		}
		err := repo.UpsertCandlestick([]Candlestick{stick})
		if err != nil {
			slog.Error("Error update candlestick", "err", err)
		}
		slog.Info("Update candlestick", "symbol", stick.Symbol, "time", stick.Time)
	}
	enableSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		enableSymbols[i] = symbol.Symbol
	}
	err = binance.SubscribeCandlestick1m(enableSymbols...)
	if err != nil {
		panic(err)
	}

	for _, symbol := range symbols {
		slog.Info("Downloading historical data", "symbol", symbol.Name)
		startTime := time.Unix(0, 0)
		candlestick, found, err := repo.GetLastCandlestick(symbol.Symbol)
		if err != nil {
			slog.Error("Error getting last candlestick", "err", err)
			continue
		}
		if found {
			startTime = candlestick.Time.Add(time.Millisecond)
		}
		for {
			sticks, err := binance.FetchCandlestick1m(symbol.Symbol, startTime)
			if err != nil {
				return
			}
			if len(sticks) == 0 {
				break
			}

			err = repo.UpsertCandlestick(sticks)
			if err != nil {
				slog.Error("Error upserting candlestick", "err", err)
				continue
			}
			slog.Info("Downloaded historical data", "symbol", symbol.Name, "startTime", startTime, "endTime", sticks[len(sticks)-1].Time)
			startTime = sticks[len(sticks)-1].Time.Add(time.Millisecond)
		}
		slog.Info("historical data up to date", "symbol", symbol.Name)
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
