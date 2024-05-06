package coinbase

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"trading-aggregator/trading"
)

type Config struct {
	URL       string
	APIKey    string
	APISecret string
}

type client struct {
	config Config
	hmac   hash.Hash
}

type sellRequest struct {
	ProductID          string             `json:"product_id"`
	Side               string             `json:"side"`
	OrderConfiguration orderConfiguration `json:"order_configuration"`
	ClientOrderID      string             `json:"client_order_id"`
}

type orderConfiguration struct {
	MarketMarketIOC marketMarketIOC `json:"market_market_ioc"`
}

type marketMarketIOC struct {
	QuoteSize string `json:"quote_size"`
	BaseSize  string `json:"base_size"`
}

type getOrderDetailResponse struct {
	Order struct {
		OrderID       string `json:"order_id"`
		ClientOrderID string `json:"client_order_id"`
		Status        string `json:"status"`
	} `json:"order"`
}

func NewClient(config Config) trading.Client {
	return &client{
		config: config,
		hmac:   hmac.New(sha256.New, []byte(config.APISecret)),
	}
}

func (c *client) Sell(req trading.SellRequest) error {
	u, err := url.Parse(c.config.URL + "/v3/brokerage/orders")
	if err != nil {
		return err
	}

	body := sellRequest{
		ProductID: fmt.Sprintf("%s-%s", req.Base, req.Quote),
		Side:      "SELL",
		OrderConfiguration: orderConfiguration{
			MarketMarketIOC: marketMarketIOC{
				BaseSize: req.Amount,
			},
		},
		ClientOrderID: req.IdempotencyKey,
	}
	bodyStr, err := json.Marshal(body)
	if err != nil {
		return err
	}

	timestamp := time.Now().Unix()
	signature := c.sign(string(bodyStr), timestamp, http.MethodPost, strings.Split(u.Path, "?")[0])

	httpReq, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(bodyStr))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("CB-ACCESS-KEY", c.config.APIKey)
	httpReq.Header.Set("CB-ACCESS-SIGN", signature)
	httpReq.Header.Set("CB-ACCESS-TIMESTAMP", strconv.FormatInt(timestamp, 10))

	res, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("get http response code %d and body %s", res.StatusCode, resBody)
	}

	fmt.Printf("get response %s\n", resBody)
	return nil
}

func (c *client) Buy(req trading.BuyRequest) error {
	return nil
}

func (c *client) GetOrderDetail(req trading.GetOrderDetailRequest) (trading.GetOrderDetailResponse, error) {
	u, err := url.Parse(c.config.URL + fmt.Sprintf("/v3/brokerage/orders/historical/%s", req.IdempotencyKey))
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	timestamp := time.Now().Unix()
	signature := c.sign("", timestamp, http.MethodGet, strings.Split(u.Path, "?")[0])

	httpReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("CB-ACCESS-KEY", c.config.APIKey)
	httpReq.Header.Set("CB-ACCESS-SIGN", signature)
	httpReq.Header.Set("CB-ACCESS-TIMESTAMP", strconv.FormatInt(timestamp, 10))

	res, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	if res.StatusCode != http.StatusOK {
		return trading.GetOrderDetailResponse{}, fmt.Errorf("get http response code %d and body %s", res.StatusCode, resBody)
	}
	fmt.Printf("get response %s\n", resBody)

	var getOrderDetailResponse getOrderDetailResponse
	err = json.Unmarshal(resBody, &getOrderDetailResponse)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	return trading.GetOrderDetailResponse{
		Status: getOrderDetailResponse.Order.Status,
	}, nil
}

func (c *client) sign(body string, timestamp int64, requestMethod, path string) string {
	c.hmac.Reset()
	c.hmac.Write([]byte(strconv.FormatInt(timestamp, 10) + requestMethod + path + body))

	return hex.EncodeToString(c.hmac.Sum(nil))
}
