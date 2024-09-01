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
	config     Config
	hmac       hash.Hash
	httpClient *http.Client
}

type orderRequest struct {
	ProductID          string             `json:"product_id"`
	Side               string             `json:"side"`
	OrderConfiguration orderConfiguration `json:"order_configuration"`
	ClientOrderID      string             `json:"client_order_id"`
}

type orderResponse struct {
	OrderID string `json:"order_id"`
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
		FilledSize    string `json:"filled_size"`
		FilledValue   string `json:"filled_value"`
	} `json:"order"`
}

func NewClient(config Config, httpClient *http.Client) trading.Client {
	return &client{
		config:     config,
		hmac:       hmac.New(sha256.New, []byte(config.APISecret)),
		httpClient: httpClient,
	}
}

func (c *client) Sell(req trading.SellRequest) (trading.SellResponse, error) {
	body := orderRequest{
		ProductID: fmt.Sprintf("%s-%s", req.Base, req.Quote),
		Side:      "SELL",
		OrderConfiguration: orderConfiguration{
			MarketMarketIOC: marketMarketIOC{
				BaseSize: req.Amount,
			},
		},
		ClientOrderID: req.ClientOrderID,
	}

	response, err := c.placeOrder(body)
	if err != nil {
		return trading.SellResponse{}, err
	}

	return trading.SellResponse{
		TradeResponse: trading.TradeResponse{
			OrderID: response.OrderID,
		},
	}, nil
}

func (c *client) Buy(req trading.BuyRequest) (trading.BuyResponse, error) {
	body := orderRequest{
		ProductID: fmt.Sprintf("%s-%s", req.Base, req.Quote),
		Side:      "BUY",
		OrderConfiguration: orderConfiguration{
			MarketMarketIOC: marketMarketIOC{
				BaseSize: req.Amount,
			},
		},
		ClientOrderID: req.ClientOrderID,
	}

	response, err := c.placeOrder(body)
	if err != nil {
		return trading.BuyResponse{}, err
	}

	return trading.BuyResponse{
		TradeResponse: trading.TradeResponse{
			OrderID: response.OrderID,
		},
	}, nil
}

func (c *client) placeOrder(req orderRequest) (orderResponse, error) {
	u, err := url.Parse(c.config.URL + "/api/v3/brokerage/orders")
	if err != nil {
		return orderResponse{}, err
	}

	bodyStr, err := json.Marshal(req)
	if err != nil {
		return orderResponse{}, err
	}

	timestamp := time.Now().Unix()
	signature := c.sign(string(bodyStr), timestamp, http.MethodPost, strings.Split(u.Path, "?")[0])

	httpReq, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(bodyStr))
	if err != nil {
		return orderResponse{}, err
	}

	httpReq.Header = c.createHeader(signature, timestamp)

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return orderResponse{}, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return orderResponse{}, err
	}

	if res.StatusCode != http.StatusOK {
		return orderResponse{}, fmt.Errorf("get http response code %d and body %s", res.StatusCode, resBody)
	}

	var response orderResponse
	err = json.Unmarshal(resBody, &response)
	if err != nil {
		return orderResponse{}, err
	}

	return response, nil
}

func (c *client) GetOrderDetail(req trading.GetOrderDetailRequest) (trading.GetOrderDetailResponse, error) {
	u, err := url.Parse(c.config.URL + fmt.Sprintf("/api/v3/brokerage/orders/historical/%s", req.IdempotencyKey))
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	timestamp := time.Now().Unix()
	signature := c.sign("", timestamp, http.MethodGet, strings.Split(u.Path, "?")[0])

	httpReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	httpReq.Header = c.createHeader(signature, timestamp)

	res, err := c.httpClient.Do(httpReq)
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

	var getOrderDetailResponse getOrderDetailResponse
	err = json.Unmarshal(resBody, &getOrderDetailResponse)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	return trading.GetOrderDetailResponse{
		Status:        getOrderDetailResponse.Order.Status,
		ExecutedBase:  getOrderDetailResponse.Order.FilledSize,
		ExecutedQuote: getOrderDetailResponse.Order.FilledValue,
	}, nil
}

func (c *client) sign(body string, timestamp int64, requestMethod, path string) string {
	c.hmac.Reset()
	c.hmac.Write([]byte(strconv.FormatInt(timestamp, 10) + requestMethod + path + body))

	return hex.EncodeToString(c.hmac.Sum(nil))
}

func (c *client) createHeader(signature string, timestamp int64) http.Header {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("CB-ACCESS-KEY", c.config.APIKey)
	header.Set("CB-ACCESS-SIGN", signature)
	header.Set("CB-ACCESS-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	return header
}
