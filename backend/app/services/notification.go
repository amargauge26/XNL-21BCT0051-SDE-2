package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/XNL-21bct0051/order-engine/pkg/types"
)

type NotificationType string

const (
	OrderExecuted    NotificationType = "ORDER_EXECUTED"
	OrderCancelled   NotificationType = "ORDER_CANCELLED"
	OrderRejected    NotificationType = "ORDER_REJECTED"
	PositionUpdated  NotificationType = "POSITION_UPDATED"
	MarginCall       NotificationType = "MARGIN_CALL"
	PriceAlert       NotificationType = "PRICE_ALERT"
)

type Notification struct {
	ID        string           `json:"id"`
	Type      NotificationType `json:"type"`
	UserID    string          `json:"user_id"`
	Message   string          `json:"message"`
	Data      interface{}     `json:"data,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	Read      bool           `json:"read"`
}

type NotificationService struct {
	logger *zap.Logger
	nc     *nats.Conn
	js     nats.JetStreamContext
}

func NewNotificationService(logger *zap.Logger, natsURL string) (*NotificationService, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create notifications stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "NOTIFICATIONS",
		Subjects: []string{"notifications.*"},
		MaxAge:   24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	return &NotificationService{
		logger: logger,
		nc:     nc,
		js:     js,
	}, nil
}

func (s *NotificationService) NotifyOrderExecuted(ctx context.Context, userID string, order *types.Order, trade *types.Trade) error {
	notification := &Notification{
		ID:        fmt.Sprintf("not_%d", time.Now().UnixNano()),
		Type:      OrderExecuted,
		UserID:    userID,
		Message:   fmt.Sprintf("Order %s executed: %v %v at %v", order.ID, trade.Quantity, order.Symbol, trade.Price),
		Data: map[string]interface{}{
			"order": order,
			"trade": trade,
		},
		CreatedAt: time.Now(),
	}

	return s.publish(notification)
}

func (s *NotificationService) NotifyOrderCancelled(ctx context.Context, userID string, order *types.Order) error {
	notification := &Notification{
		ID:        fmt.Sprintf("not_%d", time.Now().UnixNano()),
		Type:      OrderCancelled,
		UserID:    userID,
		Message:   fmt.Sprintf("Order %s cancelled", order.ID),
		Data:      order,
		CreatedAt: time.Now(),
	}

	return s.publish(notification)
}

func (s *NotificationService) NotifyOrderRejected(ctx context.Context, userID string, order *types.Order, reason string) error {
	notification := &Notification{
		ID:      fmt.Sprintf("not_%d", time.Now().UnixNano()),
		Type:    OrderRejected,
		UserID:  userID,
		Message: fmt.Sprintf("Order %s rejected: %s", order.ID, reason),
		Data: map[string]interface{}{
			"order":  order,
			"reason": reason,
		},
		CreatedAt: time.Now(),
	}

	return s.publish(notification)
}

func (s *NotificationService) NotifyMarginCall(ctx context.Context, userID string, position *Position) error {
	notification := &Notification{
		ID:      fmt.Sprintf("not_%d", time.Now().UnixNano()),
		Type:    MarginCall,
		UserID:  userID,
		Message: fmt.Sprintf("Margin call for %s position", position.Symbol),
		Data:    position,
		CreatedAt: time.Now(),
	}

	return s.publish(notification)
}

func (s *NotificationService) NotifyPriceAlert(ctx context.Context, userID, symbol string, price float64, condition string) error {
	notification := &Notification{
		ID:      fmt.Sprintf("not_%d", time.Now().UnixNano()),
		Type:    PriceAlert,
		UserID:  userID,
		Message: fmt.Sprintf("Price alert: %s %s %.2f", symbol, condition, price),
		Data: map[string]interface{}{
			"symbol":    symbol,
			"price":     price,
			"condition": condition,
		},
		CreatedAt: time.Now(),
	}

	return s.publish(notification)
}

func (s *NotificationService) SubscribeToUserNotifications(userID string, callback func(*Notification)) error {
	subject := fmt.Sprintf("notifications.%s", userID)
	
	_, err := s.js.Subscribe(subject, func(msg *nats.Msg) {
		var notification Notification
		if err := json.Unmarshal(msg.Data, &notification); err != nil {
			s.logger.Error("Failed to unmarshal notification",
				zap.Error(err),
				zap.String("user_id", userID))
			return
		}

		callback(&notification)
	})

	return err
}

func (s *NotificationService) publish(notification *Notification) error {
	subject := fmt.Sprintf("notifications.%s", notification.UserID)
	
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	_, err = s.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish notification: %w", err)
	}

	return nil
}

func (s *NotificationService) Close() error {
	return s.nc.Close()
} 