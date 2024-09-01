package bybit

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"trading-aggregator/trading"
)

var clientInstance trading.Client

func TestMain(m *testing.M) {
	clientInstance = NewClient(Config{
		URL:       "https://api.bybit.com",
		APIKey:    "",
		APISecret: "",
	}, http.DefaultClient)

	_ = m.Run()
}

func TestClient_GetOrderDetail(t *testing.T) {
	sellResp, err := clientInstance.Sell(trading.SellRequest{
		TradeRequest: trading.TradeRequest{
			Base:          "SOL",
			Quote:         "USDT",
			Amount:        "0.0001",
			ClientOrderID: uuid.NewString(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	buyResp, err := clientInstance.Buy(trading.BuyRequest{
		TradeRequest: trading.TradeRequest{
			Base:          "SOL",
			Quote:         "USDT",
			Amount:        "0.0001",
			ClientOrderID: uuid.NewString(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	sellDetail, err := clientInstance.GetOrderDetail(trading.GetOrderDetailRequest{
		Base:    "SOL",
		Quote:   "USDT",
		OrderID: sellResp.OrderID,
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(sellDetail)

	buyDetail, err := clientInstance.GetOrderDetail(trading.GetOrderDetailRequest{
		Base:    "SOL",
		Quote:   "USDT",
		OrderID: buyResp.OrderID,
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(buyDetail)
}
