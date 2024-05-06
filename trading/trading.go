package trading

type Client interface {
	Sell(SellRequest) error
	Buy(BuyRequest) error
	GetOrderDetail(GetOrderDetailRequest) (GetOrderDetailResponse, error)
}

type SellRequest struct {
	TradeRequest
}

type BuyRequest struct {
	TradeRequest
}

type TradeRequest struct {
	Base           string
	Quote          string
	Amount         string
	IdempotencyKey string
}

type GetOrderDetailRequest struct {
	Base           string
	Quote          string
	IdempotencyKey string
}

type GetOrderDetailResponse struct {
	Status string
}
