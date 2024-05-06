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
	Category    string `json:"category"`
	Symbol      string `json:"symbol"`
	Side        string `json:"side"`
	Qty         string `json:"qty"`
	OrderType   string `json:"orderType"`
	OrderLinkID string `json:"orderLinkId"`
}

type getOrderDetailRequest struct {
	Category    string `json:"category"`
	OrderLinkID string `json:"orderLinkId"`
}

func (r *getOrderDetailRequest) String() string {
	u := url.Values{}
	u["category"] = []string{r.Category}
	u["orderLinkId"] = []string{r.OrderLinkID}

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

func NewClient(config Config) trading.Client {
	return &client{
		config: config,
		hmac:   hmac.New(sha256.New, []byte(config.APISecret)),
	}
}

func (c *client) Sell(req trading.SellRequest) error {
	u, err := url.Parse(c.config.URL + "/v5/order/create")
	if err != nil {
		return err
	}

	body := sellRequest{
		Category:    "spot",
		Symbol:      fmt.Sprintf("%s%s", req.Base, req.Quote),
		Side:        "Sell",
		Qty:         req.Amount,
		OrderType:   "Market",
		OrderLinkID: req.IdempotencyKey,
	}
	bodyStr, err := json.Marshal(body)
	if err != nil {
		return err
	}

	recvWindow := int64(10000)
	timestamp := time.Now().UnixMilli()
	signature := c.sign("", string(bodyStr), timestamp, recvWindow)

	httpReq, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(bodyStr))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-BAPI-API-KEY", c.config.APIKey)
	httpReq.Header.Set("X-BAPI-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	httpReq.Header.Set("X-BAPI-SIGN", signature)
	httpReq.Header.Set("X-BAPI-SIGN-TYPE", "2")
	httpReq.Header.Set("X-BAPI-RECV-WINDOW", strconv.FormatInt(recvWindow, 10))

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
	u, err := url.Parse(c.config.URL + "/v5/order/realtime")
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	q := getOrderDetailRequest{
		Category:    "spot",
		OrderLinkID: req.IdempotencyKey,
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

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-BAPI-API-KEY", c.config.APIKey)
	httpReq.Header.Set("X-BAPI-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	httpReq.Header.Set("X-BAPI-SIGN", signature)
	httpReq.Header.Set("X-BAPI-SIGN-TYPE", "2")
	httpReq.Header.Set("X-BAPI-RECV-WINDOW", strconv.FormatInt(recvWindow, 10))

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
		Status: getOrderDetailResponse.Result.List[0].OrderStatus,
	}, nil
}

func (c *client) sign(query, body string, timestamp, recvWindow int64) string {
	c.hmac.Reset()
	c.hmac.Write([]byte(strconv.FormatInt(timestamp, 10) + c.config.APIKey + strconv.FormatInt(recvWindow, 10) + query + body))

	return hex.EncodeToString(c.hmac.Sum(nil))
}
