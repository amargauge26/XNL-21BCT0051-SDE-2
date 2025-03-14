package matching

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/XNL-21bct0051-SDE-2/order-engine/pkg/orderbook"
	"github.com/XNL-21bct0051-SDE-2/order-engine/pkg/types"
)

type MatchingEngine struct {
	orderBooks map[string]*orderbook.OrderBook
	mutex      sync.RWMutex
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		orderBooks: make(map[string]*orderbook.OrderBook),
	}
}

func (me *MatchingEngine) ProcessOrder(order *types.Order) ([]*types.Trade, error) {
	me.mutex.Lock()
	defer me.mutex.Unlock()

	// Get or create order book for symbol
	ob, exists := me.orderBooks[order.Symbol]
	if !exists {
		ob = orderbook.NewOrderBook(order.Symbol)
		me.orderBooks[order.Symbol] = ob
	}

	// Process market orders immediately
	if order.Type == types.MarketOrder {
		return me.processMarketOrder(ob, order)
	}

	// For limit orders, first check if it can be matched
	trades, err := me.matchOrder(ob, order)
	if err != nil {
		return nil, err
	}

	// If order is not fully filled, add to order book
	if order.RemainingQty > 0 {
		if err := ob.AddOrder(order); err != nil {
			return trades, err
		}
	}

	return trades, nil
}

func (me *MatchingEngine) processMarketOrder(ob *orderbook.OrderBook, order *types.Order) ([]*types.Trade, error) {
	trades, err := me.matchOrder(ob, order)
	if err != nil {
		return nil, err
	}

	// Market orders that cannot be fully filled are rejected
	if order.RemainingQty > 0 {
		order.Status = types.OrderStatusRejected
		return trades, fmt.Errorf("market order could not be fully filled")
	}

	return trades, nil
}

func (me *MatchingEngine) matchOrder(ob *orderbook.OrderBook, order *types.Order) ([]*types.Trade, error) {
	trades := make([]*types.Trade, 0)

	for order.RemainingQty > 0 {
		var matchingOrder *types.Order
		var err error

		if order.Side == types.BuyOrder {
			matchingOrder, err = ob.GetBestAsk(order.Symbol)
		} else {
			matchingOrder, err = ob.GetBestBid(order.Symbol)
		}

		if err != nil {
			return trades, err
		}

		// No matching orders
		if matchingOrder == nil {
			break
		}

		// For limit orders, check price
		if order.Type == types.LimitOrder {
			if order.Side == types.BuyOrder && order.Price < matchingOrder.Price {
				break
			}
			if order.Side == types.SellOrder && order.Price > matchingOrder.Price {
				break
			}
		}

		// Calculate trade quantity
		tradeQty := min(order.RemainingQty, matchingOrder.RemainingQty)
		tradePrice := matchingOrder.Price // Price-time priority: use existing order's price

		// Create trade
		trade := &types.Trade{
			ID:           uuid.New().String(),
			Symbol:       order.Symbol,
			Price:        tradePrice,
			Quantity:     tradeQty,
			ExecutedAt:   time.Now(),
		}

		if order.Side == types.BuyOrder {
			trade.BuyOrderID = order.ID
			trade.SellOrderID = matchingOrder.ID
			trade.BuyerUserID = order.UserID
			trade.SellerUserID = matchingOrder.UserID
		} else {
			trade.BuyOrderID = matchingOrder.ID
			trade.SellOrderID = order.ID
			trade.BuyerUserID = matchingOrder.UserID
			trade.SellerUserID = order.UserID
		}

		// Update orders
		order.FilledQty += tradeQty
		order.RemainingQty -= tradeQty
		matchingOrder.FilledQty += tradeQty
		matchingOrder.RemainingQty -= tradeQty

		// Update order statuses
		me.updateOrderStatus(order)
		me.updateOrderStatus(matchingOrder)

		// If matching order is fully filled, remove it from order book
		if matchingOrder.Status == types.OrderStatusFilled {
			if err := ob.CancelOrder(matchingOrder.ID); err != nil {
				return trades, err
			}
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

func (me *MatchingEngine) updateOrderStatus(order *types.Order) {
	if order.RemainingQty == 0 {
		order.Status = types.OrderStatusFilled
	} else if order.FilledQty > 0 {
		order.Status = types.OrderStatusPartial
	}
	order.UpdatedAt = time.Now()
}

func (me *MatchingEngine) CancelOrder(orderID string) error {
	me.mutex.Lock()
	defer me.mutex.Unlock()

	// Find order book containing the order
	for _, ob := range me.orderBooks {
		if order, err := ob.GetOrder(orderID); err == nil {
			return ob.CancelOrder(orderID)
		}
	}

	return fmt.Errorf("order %s not found", orderID)
}

func (me *MatchingEngine) GetOrderBook(symbol string) (*types.OrderBookSnapshot, error) {
	me.mutex.RLock()
	defer me.mutex.RUnlock()

	ob, exists := me.orderBooks[symbol]
	if !exists {
		return nil, fmt.Errorf("order book for symbol %s not found", symbol)
	}

	return ob.GetOrderBookSnapshot(symbol)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
} 