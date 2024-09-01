package bybit

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
	"time"

	"trading-aggregator/trading"

	"github.com/shopspring/decimal"
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
	Category    string `json:"category"`
	Symbol      string `json:"symbol"`
	Side        string `json:"side"`
	Qty         string `json:"qty"`
	OrderType   string `json:"orderType"`
	OrderLinkID string `json:"orderLinkId"`
}

type orderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		OrderID     string `json:"orderId"`
		OrderLinkID string `json:"orderLinkId"`
	} `json:"result"`
	RetExtInfo struct{} `json:"retExtInfo"`
	Time       int64    `json:"time"`
}

type getOrderDetailRequest struct {
	Category    string `json:"category"`
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

func (r *getOrderDetailRequest) String() string {
	u := url.Values{}
	u["category"] = []string{r.Category}
	if r.OrderID != "" {
		u["orderId"] = []string{r.OrderID}
	}
	if r.OrderLinkID != "" {
		u["orderLinkId"] = []string{r.OrderLinkID}
	}

	return u.Encode()
}

type getOrderDetailResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List           []orderData `json:"list"`
		NextPageCursor string      `json:"nextPageCursor"`
		Category       string      `json:"category"`
	} `json:"result"`
	RetExtInfo map[string]interface{} `json:"retExtInfo"`
	Time       int64                  `json:"time"`
}

type orderData struct {
	OrderID        string `json:"orderId"`
	OrderLinkID    string `json:"orderLinkId"`
	OrderStatus    string `json:"orderStatus"`
	AvgPrice       string `json:"avgPrice"`
	CumExecQty     string `json:"cumExecQty"`
	CumExecValue   string `json:"cumExecValue"`
	UpdatedTime    string `json:"updatedTime"`
	RejectedReason string `json:"rejectReason"`
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
		Category:    "spot",
		Symbol:      fmt.Sprintf("%s%s", req.Base, req.Quote),
		Side:        "Sell",
		Qty:         req.Amount,
		OrderType:   "Market",
		OrderLinkID: req.ClientOrderID,
	}

	response, err := c.placeOrder(body)
	if err != nil {
		return trading.SellResponse{}, err
	}

	return trading.SellResponse{
		TradeResponse: trading.TradeResponse{
			OrderID: response.Result.OrderID,
		},
	}, nil
}

func (c *client) Buy(req trading.BuyRequest) (trading.BuyResponse, error) {
	body := orderRequest{
		Category:    "spot",
		Symbol:      fmt.Sprintf("%s%s", req.Base, req.Quote),
		Side:        "Buy",
		Qty:         req.Amount,
		OrderType:   "Market",
		OrderLinkID: req.ClientOrderID,
	}

	response, err := c.placeOrder(body)
	if err != nil {
		return trading.BuyResponse{}, err
	}

	return trading.BuyResponse{
		TradeResponse: trading.TradeResponse{
			OrderID: response.Result.OrderID,
		},
	}, nil
}

func (c *client) placeOrder(req orderRequest) (orderResponse, error) {
	u, err := url.Parse(c.config.URL + "/v5/order/create")
	if err != nil {
		return orderResponse{}, err
	}

	bodyStr, err := json.Marshal(req)
	if err != nil {
		return orderResponse{}, err
	}

	recvWindow := int64(10000)
	timestamp := time.Now().UnixMilli()
	signature := c.sign("", string(bodyStr), timestamp, recvWindow)

	httpReq, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(bodyStr))
	if err != nil {
		return orderResponse{}, err
	}

	httpReq.Header = c.createHeader(signature, timestamp, recvWindow)

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
	u, err := url.Parse(c.config.URL + "/v5/order/realtime")
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	q := getOrderDetailRequest{
		Category:    "spot",
		OrderID:     req.OrderID,
		OrderLinkID: req.ClientOrderID,
	}
	qStr := q.String()
	u.RawQuery = qStr

	recvWindow := int64(10000)
	timestamp := time.Now().UnixMilli()
	signature := c.sign(qStr, "", timestamp, recvWindow)

	httpReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	httpReq.Header = c.createHeader(signature, timestamp, recvWindow)

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

	executedQuote, err := c.getExecutedQuote(getOrderDetailResponse.Result.List[0])
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	return trading.GetOrderDetailResponse{
		Status:        getOrderDetailResponse.Result.List[0].OrderStatus,
		ExecutedBase:  getOrderDetailResponse.Result.List[0].CumExecQty,
		ExecutedQuote: executedQuote.String(),
	}, nil
}

func (c *client) sign(query, body string, timestamp, recvWindow int64) string {
	c.hmac.Reset()
	c.hmac.Write([]byte(strconv.FormatInt(timestamp, 10) + c.config.APIKey + strconv.FormatInt(recvWindow, 10) + query + body))

	return hex.EncodeToString(c.hmac.Sum(nil))
}

func (c *client) createHeader(signature string, timestamp, recvWindow int64) http.Header {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("X-BAPI-API-KEY", c.config.APIKey)
	header.Set("X-BAPI-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	header.Set("X-BAPI-SIGN", signature)
	header.Set("X-BAPI-SIGN-TYPE", "2")
	header.Set("X-BAPI-RECV-WINDOW", strconv.FormatInt(recvWindow, 10))
	return header
}

func (c *client) getExecutedQuote(order orderData) (decimal.Decimal, error) {
	if order.AvgPrice != "" {
		cumExecQtc, err := decimal.NewFromString(order.CumExecQty)
		if err != nil {
			return decimal.Decimal{}, err
		}
		avgPrice, err := decimal.NewFromString(order.AvgPrice)
		if err != nil {
			return decimal.Decimal{}, err
		}
		return cumExecQtc.Mul(avgPrice), nil
	}

	return decimal.NewFromString(order.CumExecValue)
}
