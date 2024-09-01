package trading

type Client interface {
	Sell(SellRequest) (SellResponse, error)
	Buy(BuyRequest) (BuyResponse, error)
	GetOrderDetail(GetOrderDetailRequest) (GetOrderDetailResponse, error)
}

type SellRequest struct {
	TradeRequest
}

type SellResponse struct {
	TradeResponse
}

type BuyRequest struct {
	TradeRequest
}

type BuyResponse struct {
	TradeResponse
}

type TradeRequest struct {
	Base          string
	Quote         string
	Amount        string
	ClientOrderID string
}

type TradeResponse struct {
	OrderID string
}

type GetOrderDetailRequest struct {
	Base          string
	Quote         string
	OrderID       string
	ClientOrderID string
}

type GetOrderDetailResponse struct {
	Status        string
	ExecutedBase  string
	ExecutedQuote string
}
