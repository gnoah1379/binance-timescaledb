package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SpotBinance struct {
	testnet             bool
	httpclient          http.Client
	ws                  *websocket.Conn
	candlestickCallback func(candlestick WsKline)
	subscribed          map[string]bool
	mu                  sync.Mutex
}

func NewSpotBinance() *SpotBinance {
	return &SpotBinance{
		httpclient: http.Client{
			Timeout: 5 * time.Second,
		},
		subscribed: make(map[string]bool),
	}
}

func (b *SpotBinance) GetBaseAPI() string {
	if b.testnet {
		return "https://testnet.binance.vision"
	}
	return "https://api.binance.com"
}

func (b *SpotBinance) GetStreamUrl() string {
	if b.testnet {
		return "wss://testnet.binance.vision/stream"
	}
	return "wss://stream.binance.com:9443/stream"

}

func SafeParseFloat(s any) float64 {
	switch v := s.(type) {
	case float64:
		return v
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

func (b *SpotBinance) FetchCandlestick1m(symbol string, start time.Time) ([]Candlestick, error) {
	uri := "/api/v3/klines"
	req, err := http.NewRequest(http.MethodGet, b.GetBaseAPI()+uri, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("symbol", symbol)
	q.Add("interval", "1m")
	q.Add("startTime", strconv.FormatInt(start.UnixMilli(), 10))
	q.Add("limit", "1000")
	req.URL.RawQuery = q.Encode()
	resp, err := b.httpclient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var raw [][]any
	err = json.NewDecoder(resp.Body).Decode(&raw)
	if err != nil {
		return nil, err
	}

	var candlesticks = make([]Candlestick, len(raw))
	for i, row := range raw {
		candlesticks[i] = Candlestick{
			Time:   time.UnixMilli(int64(row[0].(float64))),
			Symbol: symbol,
			Open:   SafeParseFloat(row[1]),
			High:   SafeParseFloat(row[2]),
			Low:    SafeParseFloat(row[3]),
			Close:  SafeParseFloat(row[4]),
			Volume: SafeParseFloat(row[5]),
		}
	}
	return candlesticks, nil
}

func (b *SpotBinance) SetTestnet(testnet bool) {
	b.testnet = testnet
}

func (b *SpotBinance) getSubscribed() []string {
	var result []string
	for k, v := range b.subscribed {
		if v {
			result = append(result, k)
		}
	}
	return result
}

func (b *SpotBinance) ConnectStream() error {
	ws, _, err := websocket.DefaultDialer.Dial(b.GetStreamUrl(), nil)
	if err != nil {
		slog.Error("connect stream error", "err", err)
		return err
	}
	b.mu.Lock()
	b.ws = ws
	subscribed := b.getSubscribed()
	b.mu.Unlock()
	err = b.SubscribeCandlestick1m(subscribed...)
	if err != nil {
		return err
	}
	go b.handleStreamMessage()
	return nil
}

type StreamData[T any] struct {
	Stream string `json:"stream"`
	Data   T      `json:"data"`
}

// WsKlineEvent define websocket kline event
type WsKlineEvent struct {
	Event  string  `json:"e"`
	Time   int64   `json:"E"`
	Symbol string  `json:"s"`
	Kline  WsKline `json:"k"`
}

// WsKline define websocket kline
type WsKline struct {
	StartTime            int64  `json:"t"`
	EndTime              int64  `json:"T"`
	Symbol               string `json:"s"`
	Interval             string `json:"i"`
	FirstTradeID         int64  `json:"f"`
	LastTradeID          int64  `json:"L"`
	Open                 string `json:"o"`
	Close                string `json:"c"`
	High                 string `json:"h"`
	Low                  string `json:"l"`
	Volume               string `json:"v"`
	TradeNum             int64  `json:"n"`
	IsFinal              bool   `json:"x"`
	QuoteVolume          string `json:"q"`
	ActiveBuyVolume      string `json:"V"`
	ActiveBuyQuoteVolume string `json:"Q"`
}

func (b *SpotBinance) handleStreamMessage() {
	for {
		b.mu.Lock()
		_, message, err := b.ws.ReadMessage()
		b.mu.Unlock()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("read stream error", "err", err)
				err = b.ConnectStream()
				if err != nil {
					slog.Error("reconnect stream error", "err", err)
				}
				return
			}
			continue
		}
		if bytes.Equal(message, []byte(`{"result":null,"id":1}`)) {
			continue
		}
		var msg StreamData[WsKlineEvent]
		err = json.Unmarshal(message, &msg)
		if err != nil {
			slog.Error("unmarshal stream message error", "err", err)
			continue
		}
		data := msg.Data
		if data.Event != "kline" {
			continue
		}

		b.candlestickCallback(data.Kline)
	}
}

func (b *SpotBinance) SubscribeCandlestick1m(symbols ...string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var params []string
	for _, symbol := range symbols {
		b.subscribed[symbol] = true
		params = append(params, strings.ToLower(symbol)+"@kline_1m")
	}
	msg := map[string]any{
		"method": "SUBSCRIBE",
		"params": params,
		"id":     1,
	}
	return b.ws.WriteJSON(msg)
}
