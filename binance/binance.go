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

type sellRequest struct {
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`
	Quantity      string `json:"quantity"`
	Type          string `json:"type"`
	ClientOrderID string `json:"newClientOrderId"`
	Timestamp     int64  `json:"timestamp"`
}

func (r *sellRequest) String() string {
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
}

type getOrderDetailResponse struct {
	Symbol        string `json:"symbol"`
	ClientOrderID string `json:"clientOrderId"`
	Status        string `json:"status"`
}

func (r *getOrderDetailRequest) String() string {
	u := url.Values{}
	u["symbol"] = []string{r.Symbol}
	u["origClientOrderId"] = []string{r.ClientOrderID}

	return u.Encode()
}

type client struct {
	config Config
	hmac   hash.Hash
}

func NewClient(config Config) trading.Client {
	return &client{
		config: config,
		hmac:   hmac.New(sha256.New, []byte(config.APISecret)),
	}
}

func (c *client) Sell(req trading.SellRequest) error {
	u, err := url.Parse(c.config.URL + "/v3/order")
	if err != nil {
		return err
	}

	body := sellRequest{
		Symbol:        fmt.Sprintf("%s%s", req.Base, req.Quote),
		Side:          "SELL",
		Quantity:      req.Amount,
		Type:          "MARKET",
		ClientOrderID: req.IdempotencyKey,
		Timestamp:     time.Now().UTC().UnixMilli(),
	}
	bodyStr := body.String()

	signature := c.sign("", bodyStr)

	query := u.Query()
	query.Add("signature", signature)
	u.RawQuery = query.Encode()

	httpReq, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(bodyStr))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-MBX-APIKEY", c.config.APIKey)

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
	u, err := url.Parse(c.config.URL + "/v3/order")
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	q := getOrderDetailRequest{
		Symbol:        fmt.Sprintf("%s%s", req.Base, req.Quote),
		ClientOrderID: req.IdempotencyKey,
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

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-MBX-APIKEY", c.config.APIKey)

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

	var getOrderStatusResponse getOrderDetailResponse
	err = json.Unmarshal(resBody, &getOrderStatusResponse)
	if err != nil {
		return trading.GetOrderDetailResponse{}, err
	}

	return trading.GetOrderDetailResponse{
		Status: getOrderStatusResponse.Status,
	}, nil
}

func (c *client) sign(query, body string) string {
	c.hmac.Reset()
	c.hmac.Write([]byte(query + body))

	return hex.EncodeToString(c.hmac.Sum(nil))
}
