package orderbook

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/XNL-21bct0051-SDE-2/order-engine/pkg/types"
)

type priceLevel struct {
	price   float64
	orders  []*types.Order
	volume  float64
}

type priceLevels []*priceLevel

func (pl priceLevels) Len() int           { return len(pl) }
func (pl priceLevels) Swap(i, j int)      { pl[i], pl[j] = pl[j], pl[i] }
func (pl priceLevels) Push(x interface{}) { pl = append(pl, x.(*priceLevel)) }
func (pl priceLevels) Pop() interface{} {
	old := pl
	n := len(old)
	x := old[n-1]
	pl = old[0 : n-1]
	return x
}

// BuyPriceLevels implements max heap for buy orders (highest price first)
type buyPriceLevels struct{ priceLevels }

func (bpl buyPriceLevels) Less(i, j int) bool {
	return bpl.priceLevels[i].price > bpl.priceLevels[j].price
}

// SellPriceLevels implements min heap for sell orders (lowest price first)
type sellPriceLevels struct{ priceLevels }

func (spl sellPriceLevels) Less(i, j int) bool {
	return spl.priceLevels[i].price < spl.priceLevels[j].price
}

type OrderBook struct {
	symbol string
	bids   *buyPriceLevels
	asks   *sellPriceLevels
	orders map[string]*types.Order
	mutex  sync.RWMutex
}

func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		symbol: symbol,
		bids:   &buyPriceLevels{make(priceLevels, 0)},
		asks:   &sellPriceLevels{make(priceLevels, 0)},
		orders: make(map[string]*types.Order),
	}
}

func (ob *OrderBook) AddOrder(order *types.Order) error {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	if _, exists := ob.orders[order.ID]; exists {
		return fmt.Errorf("order %s already exists", order.ID)
	}

	// Initialize order
	order.Status = types.OrderStatusNew
	order.RemainingQty = order.Quantity
	order.FilledQty = 0
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	// Add to orders map
	ob.orders[order.ID] = order

	// Add to price level
	var levels *priceLevels
	if order.Side == types.BuyOrder {
		levels = &ob.bids.priceLevels
	} else {
		levels = &ob.asks.priceLevels
	}

	// Find or create price level
	var level *priceLevel
	for _, l := range *levels {
		if l.price == order.Price {
			level = l
			break
		}
	}

	if level == nil {
		level = &priceLevel{
			price:  order.Price,
			orders: make([]*types.Order, 0),
		}
		heap.Push(levels, level)
	}

	level.orders = append(level.orders, order)
	level.volume += order.RemainingQty

	return nil
}

func (ob *OrderBook) CancelOrder(orderID string) error {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	order, exists := ob.orders[orderID]
	if !exists {
		return fmt.Errorf("order %s not found", orderID)
	}

	if order.Status == types.OrderStatusCancelled {
		return fmt.Errorf("order %s already cancelled", orderID)
	}

	// Update order status
	order.Status = types.OrderStatusCancelled
	order.UpdatedAt = time.Now()

	// Remove from price level
	var levels *priceLevels
	if order.Side == types.BuyOrder {
		levels = &ob.bids.priceLevels
	} else {
		levels = &ob.asks.priceLevels
	}

	for _, level := range *levels {
		if level.price == order.Price {
			for i, o := range level.orders {
				if o.ID == orderID {
					level.orders = append(level.orders[:i], level.orders[i+1:]...)
					level.volume -= order.RemainingQty
					break
				}
			}
			break
		}
	}

	return nil
}

func (ob *OrderBook) GetOrder(orderID string) (*types.Order, error) {
	ob.mutex.RLock()
	defer ob.mutex.RUnlock()

	order, exists := ob.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	return order, nil
}

func (ob *OrderBook) GetOrdersByUser(userID string) ([]*types.Order, error) {
	ob.mutex.RLock()
	defer ob.mutex.RUnlock()

	orders := make([]*types.Order, 0)
	for _, order := range ob.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}

	return orders, nil
}

func (ob *OrderBook) GetOrdersBySymbol(symbol string) ([]*types.Order, error) {
	if symbol != ob.symbol {
		return nil, fmt.Errorf("invalid symbol %s", symbol)
	}

	ob.mutex.RLock()
	defer ob.mutex.RUnlock()

	orders := make([]*types.Order, 0, len(ob.orders))
	for _, order := range ob.orders {
		orders = append(orders, order)
	}

	return orders, nil
}

func (ob *OrderBook) GetBestBid(symbol string) (*types.Order, error) {
	if symbol != ob.symbol {
		return nil, fmt.Errorf("invalid symbol %s", symbol)
	}

	ob.mutex.RLock()
	defer ob.mutex.RUnlock()

	if len(ob.bids.priceLevels) == 0 {
		return nil, nil
	}

	level := ob.bids.priceLevels[0]
	if len(level.orders) == 0 {
		return nil, nil
	}

	return level.orders[0], nil
}

func (ob *OrderBook) GetBestAsk(symbol string) (*types.Order, error) {
	if symbol != ob.symbol {
		return nil, fmt.Errorf("invalid symbol %s", symbol)
	}

	ob.mutex.RLock()
	defer ob.mutex.RUnlock()

	if len(ob.asks.priceLevels) == 0 {
		return nil, nil
	}

	level := ob.asks.priceLevels[0]
	if len(level.orders) == 0 {
		return nil, nil
	}

	return level.orders[0], nil
}

func (ob *OrderBook) GetOrderBookSnapshot(symbol string) (*types.OrderBookSnapshot, error) {
	if symbol != ob.symbol {
		return nil, fmt.Errorf("invalid symbol %s", symbol)
	}

	ob.mutex.RLock()
	defer ob.mutex.RUnlock()

	snapshot := &types.OrderBookSnapshot{
		Symbol:    ob.symbol,
		Timestamp: time.Now(),
		Bids:      make([]types.OrderBookLevel, 0, len(ob.bids.priceLevels)),
		Asks:      make([]types.OrderBookLevel, 0, len(ob.asks.priceLevels)),
	}

	// Add bids
	for _, level := range ob.bids.priceLevels {
		if len(level.orders) > 0 {
			snapshot.Bids = append(snapshot.Bids, types.OrderBookLevel{
				Price:    level.price,
				Quantity: level.volume,
				Orders:   len(level.orders),
			})
		}
	}

	// Add asks
	for _, level := range ob.asks.priceLevels {
		if len(level.orders) > 0 {
			snapshot.Asks = append(snapshot.Asks, types.OrderBookLevel{
				Price:    level.price,
				Quantity: level.volume,
				Orders:   len(level.orders),
			})
		}
	}

	return snapshot, nil
} 