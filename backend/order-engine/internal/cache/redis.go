package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/XNL-21bct0051-SDE-2/order-engine/pkg/types"
)

type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisCache(addr, password string, db int, logger *zap.Logger) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		logger: logger,
	}, nil
}

// Key prefixes
const (
	orderBookPrefix = "orderbook:"
	tradePrefix     = "trade:"
	orderPrefix     = "order:"
)

// CacheOrderBook stores the order book snapshot in Redis
func (c *RedisCache) CacheOrderBook(ctx context.Context, symbol string, snapshot *types.OrderBookSnapshot) error {
	key := orderBookPrefix + symbol
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal order book: %w", err)
	}

	// Store with expiration of 1 minute
	if err := c.client.Set(ctx, key, data, time.Minute).Err(); err != nil {
		return fmt.Errorf("failed to cache order book: %w", err)
	}

	c.logger.Debug("Cached order book",
		zap.String("symbol", symbol),
		zap.Time("timestamp", snapshot.Timestamp))

	return nil
}

// GetOrderBook retrieves the order book snapshot from Redis
func (c *RedisCache) GetOrderBook(ctx context.Context, symbol string) (*types.OrderBookSnapshot, error) {
	key := orderBookPrefix + symbol
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get order book from cache: %w", err)
	}

	var snapshot types.OrderBookSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order book: %w", err)
	}

	return &snapshot, nil
}

// CacheTrade stores the trade in Redis
func (c *RedisCache) CacheTrade(ctx context.Context, trade *types.Trade) error {
	key := fmt.Sprintf("%s%s:%s", tradePrefix, trade.Symbol, trade.ID)
	data, err := json.Marshal(trade)
	if err != nil {
		return fmt.Errorf("failed to marshal trade: %w", err)
	}

	// Store with expiration of 1 hour
	if err := c.client.Set(ctx, key, data, time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to cache trade: %w", err)
	}

	// Add to sorted set for time-based queries
	score := float64(trade.ExecutedAt.Unix())
	zkey := tradePrefix + trade.Symbol
	if err := c.client.ZAdd(ctx, zkey, &redis.Z{
		Score:  score,
		Member: trade.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add trade to sorted set: %w", err)
	}

	c.logger.Debug("Cached trade",
		zap.String("symbol", trade.Symbol),
		zap.String("trade_id", trade.ID))

	return nil
}

// GetRecentTrades retrieves recent trades for a symbol
func (c *RedisCache) GetRecentTrades(ctx context.Context, symbol string, limit int) ([]*types.Trade, error) {
	zkey := tradePrefix + symbol
	tradeIDs, err := c.client.ZRevRange(ctx, zkey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent trade IDs: %w", err)
	}

	if len(tradeIDs) == 0 {
		return nil, nil
	}

	trades := make([]*types.Trade, 0, len(tradeIDs))
	for _, id := range tradeIDs {
		key := fmt.Sprintf("%s%s:%s", tradePrefix, symbol, id)
		data, err := c.client.Get(ctx, key).Bytes()
		if err != nil {
			if err != redis.Nil {
				c.logger.Error("Failed to get trade from cache",
					zap.Error(err),
					zap.String("trade_id", id))
			}
			continue
		}

		var trade types.Trade
		if err := json.Unmarshal(data, &trade); err != nil {
			c.logger.Error("Failed to unmarshal trade",
				zap.Error(err),
				zap.String("trade_id", id))
			continue
		}

		trades = append(trades, &trade)
	}

	return trades, nil
}

// CacheOrder stores the order in Redis
func (c *RedisCache) CacheOrder(ctx context.Context, order *types.Order) error {
	key := orderPrefix + order.ID
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	// Store with expiration of 24 hours
	if err := c.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to cache order: %w", err)
	}

	c.logger.Debug("Cached order",
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID))

	return nil
}

// GetOrder retrieves an order from Redis
func (c *RedisCache) GetOrder(ctx context.Context, orderID string) (*types.Order, error) {
	key := orderPrefix + orderID
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get order from cache: %w", err)
	}

	var order types.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}

	return &order, nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
} 