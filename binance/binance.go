package binance

import (
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

type placeOrderRequest struct {
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`
	Quantity      string `json:"quantity"`
	Type          string `json:"type"`
	ClientOrderID string `json:"newClientOrderId"`
	Timestamp     int64  `json:"timestamp"`
}

type placeOrderResponse struct {
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

func (r *placeOrderRequest) String() string {
	u := url.Values{}
	u["symbol"] = []string{r.Symbol}
	u["side"] = []string{r.Side}
	u["type"] = []string{r.Type}
	u["quantity"] = []string{r.Quantity}
	u["newClientOrderId"] = []string{r.ClientOrderID}
	u["timestamp"] = []string{strconv.FormatInt(r.Timestamp, 10)}

	return u.Encode()
}

type getOrderDetailRequest struct {
	Symbol        string `json:"symbol"`
	ClientOrderID string `json:"origClientOrderId"`
	OrderID       string `json:"orderId"`
	Timestamp     int64  `json:"timestamp"`
}

type getOrderDetailResponse struct {
	Symbol              string `json:"symbol"`
	ClientOrderID       string `json:"clientOrderId"`
	Status              string `json:"status"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
}

func (r *getOrderDetailRequest) String() string {
	u := url.Values{}
	u["symbol"] = []string{r.Symbol}
	if r.ClientOrderID != "" {
		u["origClientOrderId"] = []string{r.ClientOrderID}
	}
	if r.OrderID != "" {
		u["orderId"] = []string{r.OrderID}
	}
	u["timestamp"] = []string{strconv.FormatInt(r.Timestamp, 10)}

	return u.Encode()
}

type errorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type client struct {
	config     Config
	hmac       hash.Hash
	httpClient *http.Client
}

func NewClient(config Config, httpClient *http.Client) trading.Client {
	return &client{
		config:     config,
		hmac:       hmac.New(sha256.New, []byte(config.APISecret)),
		httpClient: httpClient,
	}
}

func (c *client) Sell(req trading.SellRequest) (trading.SellResponse, error) {
	body := placeOrderRequest{
		Symbol:        fmt.Sprintf("%s%s", req.Base, req.Quote),
		Side:          "SELL",
		Quantity:      req.Amount,
		Type:          "MARKET",
		ClientOrderID: req.ClientOrderID,
		Timestamp:     time.Now().UTC().UnixMilli(),
	}

	sellResponse, err := c.placeOrder(body)
	if err != nil {
		return trading.SellResponse{}, err
	}

	return trading.SellResponse{
		TradeResponse: trading.TradeResponse{
			OrderID: strconv.FormatInt(sellResponse.OrderID, 10),
		},
	}, nil
}

func (c *client) Buy(req trading.BuyRequest) (trading.BuyResponse, error) {
	body := placeOrderRequest{
		Symbol:        fmt.Sprintf("%s%s", req.Base, req.Quote),
		Side:          "BUY",
		Quantity:      req.Amount,
		Type:          "MARKET",
		ClientOrderID: req.ClientOrderID,
		Timestamp:     time.Now().UTC().UnixMilli(),
	}

	sellResponse, err := c.placeOrder(body)
	if err != nil {
		return trading.BuyResponse{}, err
	}

	return trading.BuyResponse{
		TradeResponse: trading.TradeResponse{
			OrderID: strconv.FormatInt(sellResponse.OrderID, 10),
		},
	}, nil
}

func (c *client) placeOrder(req placeOrderRequest) (placeOrderResponse, error) {
	u, err := url.Parse(c.config.URL + "/api/v3/order")
	if err != nil {
		return placeOrderResponse{}, err
	}

	bodyStr := req.String()

	signature := c.sign("", bodyStr)

	query := u.Query()
	query.Add("signature", signature)
	u.RawQuery = query.Encode()

	httpReq, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(bodyStr))
	if err != nil {
		return placeOrderResponse{}, err
	}

	httpReq.Header = c.createHeader()

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return placeOrderResponse{}, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return placeOrderResponse{}, err
	}

	if res.StatusCode != http.StatusOK {
		var errorResponse errorResponse
		err = json.Unmarshal(resBody, &errorResponse)
		if err == nil {
			return placeOrderResponse{}, fmt.Errorf("get http response code %d and error %v", res.StatusCode, errorResponse)
		}

		return placeOrderResponse{}, fmt.Errorf("get http response code %d and body %s", res.StatusCode, resBody)
	}

	var response placeOrderResponse
	err = json.Unmarshal(resBody, &response)
	if err != nil {
		return placeOrderResponse{}, err
	}

	return response, nil
}

func (c *client) GetOrderDetail(req trading.GetOrderDetailRequest) (trading.GetOrderDetailResponse, error) {
	u, err := url.Parse(c.config.URL + "/api/v3/order")
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	q := getOrderDetailRequest{
		Symbol:        fmt.Sprintf("%s%s", req.Base, req.Quote),
		OrderID:       req.OrderID,
		ClientOrderID: req.ClientOrderID,
		Timestamp:     time.Now().UTC().UnixMilli(),
	}
	qStr := q.String()
	u.RawQuery = qStr

	signature := c.sign(qStr, "")

	query := u.Query()
	query.Add("signature", signature)
	u.RawQuery = query.Encode()

	httpReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	httpReq.Header = c.createHeader()

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

	var getOrderStatusResponse getOrderDetailResponse
	err = json.Unmarshal(resBody, &getOrderStatusResponse)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	return trading.GetOrderDetailResponse{
		Status:        getOrderStatusResponse.Status,
		ExecutedBase:  getOrderStatusResponse.ExecutedQty,
		ExecutedQuote: getOrderStatusResponse.CummulativeQuoteQty,
	}, nil
}

func (c *client) sign(query, body string) string {
	c.hmac.Reset()
	c.hmac.Write([]byte(query + body))

	return hex.EncodeToString(c.hmac.Sum(nil))
}

func (c *client) createHeader() http.Header {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("X-MBX-APIKEY", c.config.APIKey)
	return header
}
