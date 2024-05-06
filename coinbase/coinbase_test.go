package coinbase

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"trading-aggregator/trading"
)

var clientInstance trading.Client

func TestMain(m *testing.M) {
	clientInstance = NewClient(Config{
		URL:       "https://api.coinbase.com/api",
		APIKey:    "",
		APISecret: "",
	})

	_ = m.Run()
}

func TestClient_Sell(t *testing.T) {
	err := clientInstance.Sell(trading.SellRequest{
		TradeRequest: trading.TradeRequest{
			Base:           "SOL",
			Quote:          "USDT",
			Amount:         "0.0001",
			IdempotencyKey: uuid.NewString(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_GetOrderDetail(t *testing.T) {
	resp, err := clientInstance.GetOrderDetail(trading.GetOrderDetailRequest{
		Base:           "SOL",
		Quote:          "USDT",
		IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(resp)
}
