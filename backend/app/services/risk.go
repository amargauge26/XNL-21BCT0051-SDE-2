package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/XNL-21bct0051/order-engine/pkg/types"
)

type RiskLimits struct {
	MaxOrderValue     float64 `json:"max_order_value"`
	MaxPositionValue  float64 `json:"max_position_value"`
	MaxLeverage      float64 `json:"max_leverage"`
	MinMarginRatio   float64 `json:"min_margin_ratio"`
}

type Position struct {
	Symbol    string    `json:"symbol"`
	Quantity  float64   `json:"quantity"`
	AvgPrice  float64   `json:"avg_price"`
	Value     float64   `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RiskService struct {
	logger      *zap.Logger
	marketData  *MarketDataService
	positions   sync.Map
	userLimits  sync.Map
	defaultLimits RiskLimits
}

func NewRiskService(logger *zap.Logger, marketData *MarketDataService) *RiskService {
	return &RiskService{
		logger:     logger,
		marketData: marketData,
		defaultLimits: RiskLimits{
			MaxOrderValue:    100000,  // $100k
			MaxPositionValue: 1000000, // $1M
			MaxLeverage:     5,        // 5x
			MinMarginRatio:  0.2,      // 20%
		},
	}
}

func (s *RiskService) ValidateOrder(ctx context.Context, order *types.Order) error {
	// Get current market price
	price, err := s.marketData.GetPrice(ctx, order.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get market price: %w", err)
	}

	// Calculate order value
	orderValue := order.Quantity * price.Price

	// Get user limits
	limits := s.getUserLimits(order.UserID)

	// Check order value limit
	if orderValue > limits.MaxOrderValue {
		return fmt.Errorf("order value %.2f exceeds limit %.2f", orderValue, limits.MaxOrderValue)
	}

	// Get current position
	position := s.getPosition(order.UserID, order.Symbol)

	// Calculate new position after order
	newQuantity := position.Quantity
	if order.Side == types.BuyOrder {
		newQuantity += order.Quantity
	} else {
		newQuantity -= order.Quantity
	}

	// Calculate new position value
	newValue := newQuantity * price.Price

	// Check position value limit
	if newValue > limits.MaxPositionValue {
		return fmt.Errorf("position value %.2f would exceed limit %.2f", newValue, limits.MaxPositionValue)
	}

	return nil
}

func (s *RiskService) UpdatePosition(userID, symbol string, trade *types.Trade) {
	key := fmt.Sprintf("%s:%s", userID, symbol)
	
	// Get current position
	pos, _ := s.positions.LoadOrStore(key, &Position{
		Symbol: symbol,
	})
	position := pos.(*Position)

	// Update position
	if trade.BuyerUserID == userID {
		position.Quantity += trade.Quantity
		position.Value += trade.Quantity * trade.Price
	} else if trade.SellerUserID == userID {
		position.Quantity -= trade.Quantity
		position.Value -= trade.Quantity * trade.Price
	}

	// Update average price
	if position.Quantity != 0 {
		position.AvgPrice = position.Value / position.Quantity
	}
	position.UpdatedAt = time.Now()

	s.positions.Store(key, position)
}

func (s *RiskService) GetPosition(userID, symbol string) *Position {
	return s.getPosition(userID, symbol)
}

func (s *RiskService) SetUserLimits(userID string, limits RiskLimits) {
	s.userLimits.Store(userID, limits)
}

func (s *RiskService) getPosition(userID, symbol string) *Position {
	key := fmt.Sprintf("%s:%s", userID, symbol)
	if pos, ok := s.positions.Load(key); ok {
		return pos.(*Position)
	}
	return &Position{Symbol: symbol}
}

func (s *RiskService) getUserLimits(userID string) RiskLimits {
	if limits, ok := s.userLimits.Load(userID); ok {
		return limits.(RiskLimits)
	}
	return s.defaultLimits
}

func (s *RiskService) CalculateMarginRequirement(position *Position, price float64) float64 {
	return position.Value * s.defaultLimits.MinMarginRatio
} 