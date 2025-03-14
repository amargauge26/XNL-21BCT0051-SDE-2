package types

import (
	"time"
)

type OrderType string
type OrderSide string
type OrderStatus string

const (
	LimitOrder  OrderType = "LIMIT"
	MarketOrder OrderType = "MARKET"
	StopOrder   OrderType = "STOP"

	BuyOrder  OrderSide = "BUY"
	SellOrder OrderSide = "SELL"

	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusPartial   OrderStatus = "PARTIAL"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRejected  OrderStatus = "REJECTED"
)

type Order struct {
	ID            string      `json:"id"`
	UserID        string      `json:"user_id"`
	Symbol        string      `json:"symbol"`
	Type         OrderType   `json:"type"`
	Side         OrderSide   `json:"side"`
	Price        float64     `json:"price"`
	Quantity     float64     `json:"quantity"`
	FilledQty    float64     `json:"filled_qty"`
	RemainingQty float64     `json:"remaining_qty"`
	Status       OrderStatus `json:"status"`
	StopPrice    float64     `json:"stop_price,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type Trade struct {
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	BuyOrderID   string    `json:"buy_order_id"`
	SellOrderID  string    `json:"sell_order_id"`
	Price        float64   `json:"price"`
	Quantity     float64   `json:"quantity"`
	ExecutedAt   time.Time `json:"executed_at"`
	BuyerUserID  string    `json:"buyer_user_id"`
	SellerUserID string    `json:"seller_user_id"`
}

type OrderBook interface {
	AddOrder(order *Order) error
	CancelOrder(orderID string) error
	GetOrder(orderID string) (*Order, error)
	GetOrdersByUser(userID string) ([]*Order, error)
	GetOrdersBySymbol(symbol string) ([]*Order, error)
	GetBestBid(symbol string) (*Order, error)
	GetBestAsk(symbol string) (*Order, error)
	GetOrderBookSnapshot(symbol string) (*OrderBookSnapshot, error)
}

type OrderBookSnapshot struct {
	Symbol    string          `json:"symbol"`
	Timestamp time.Time       `json:"timestamp"`
	Bids      []OrderBookLevel `json:"bids"`
	Asks      []OrderBookLevel `json:"asks"`
}

type OrderBookLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Orders   int     `json:"orders"`
}

type MatchingEngine interface {
	ProcessOrder(order *Order) ([]*Trade, error)
	CancelOrder(orderID string) error
	GetOrderBook(symbol string) (*OrderBookSnapshot, error)
} 