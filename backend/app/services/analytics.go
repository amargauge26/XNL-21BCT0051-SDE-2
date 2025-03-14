package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/XNL-21bct0051/order-engine/pkg/types"
)

type TimeFrame string

const (
	TimeFrame1m  TimeFrame = "1m"
	TimeFrame5m  TimeFrame = "5m"
	TimeFrame15m TimeFrame = "15m"
	TimeFrame1h  TimeFrame = "1h"
	TimeFrame4h  TimeFrame = "4h"
	TimeFrame1d  TimeFrame = "1d"
)

type OHLCV struct {
	Symbol    string    `json:"symbol"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

type VolumeProfile struct {
	Symbol    string             `json:"symbol"`
	TimeFrame TimeFrame         `json:"time_frame"`
	Levels    []VolumePriceLevel `json:"levels"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
}

type VolumePriceLevel struct {
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	BuyVolume float64 `json:"buy_volume"`
	SellVolume float64 `json:"sell_volume"`
	Trades    int     `json:"trades"`
}

type AnalyticsService struct {
	logger     *zap.Logger
	marketData *MarketDataService
	cache      sync.Map
}

func NewAnalyticsService(logger *zap.Logger, marketData *MarketDataService) *AnalyticsService {
	return &AnalyticsService{
		logger:     logger,
		marketData: marketData,
	}
}

func (s *AnalyticsService) CalculateOHLCV(trades []*types.Trade) *OHLCV {
	if len(trades) == 0 {
		return nil
	}

	ohlcv := &OHLCV{
		Symbol:    trades[0].Symbol,
		Open:      trades[0].Price,
		High:      trades[0].Price,
		Low:       trades[0].Price,
		Close:     trades[len(trades)-1].Price,
		Timestamp: trades[0].ExecutedAt,
	}

	for _, trade := range trades {
		ohlcv.High = math.Max(ohlcv.High, trade.Price)
		ohlcv.Low = math.Min(ohlcv.Low, trade.Price)
		ohlcv.Volume += trade.Quantity
	}

	return ohlcv
}

func (s *AnalyticsService) CalculateVWAP(trades []*types.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}

	var volumeSum, priceVolumeSum float64
	for _, trade := range trades {
		volumeSum += trade.Quantity
		priceVolumeSum += trade.Price * trade.Quantity
	}

	if volumeSum == 0 {
		return 0
	}

	return priceVolumeSum / volumeSum
}

func (s *AnalyticsService) GenerateVolumeProfile(trades []*types.Trade, numLevels int) *VolumeProfile {
	if len(trades) == 0 {
		return nil
	}

	// Find price range
	minPrice, maxPrice := trades[0].Price, trades[0].Price
	for _, trade := range trades {
		minPrice = math.Min(minPrice, trade.Price)
		maxPrice = math.Max(maxPrice, trade.Price)
	}

	// Calculate price levels
	priceStep := (maxPrice - minPrice) / float64(numLevels)
	levels := make(map[float64]*VolumePriceLevel)

	for _, trade := range trades {
		levelPrice := math.Floor((trade.Price-minPrice)/priceStep) * priceStep + minPrice
		level, exists := levels[levelPrice]
		if !exists {
			level = &VolumePriceLevel{
				Price: levelPrice,
			}
			levels[levelPrice] = level
		}

		level.Volume += trade.Quantity
		level.Trades++
		if trade.BuyOrderID != "" {
			level.BuyVolume += trade.Quantity
		} else {
			level.SellVolume += trade.Quantity
		}
	}

	// Convert map to sorted slice
	result := make([]VolumePriceLevel, 0, len(levels))
	for _, level := range levels {
		result = append(result, *level)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Price < result[j].Price
	})

	return &VolumeProfile{
		Symbol:    trades[0].Symbol,
		TimeFrame: TimeFrame1h, // Default timeframe
		Levels:    result,
		StartTime: trades[0].ExecutedAt,
		EndTime:   trades[len(trades)-1].ExecutedAt,
	}
}

func (s *AnalyticsService) CalculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// Calculate returns
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = math.Log(prices[i] / prices[i-1])
	}

	// Calculate standard deviation
	var sum, sumSquared float64
	for _, r := range returns {
		sum += r
		sumSquared += r * r
	}
	mean := sum / float64(len(returns))
	variance := sumSquared/float64(len(returns)) - mean*mean
	
	return math.Sqrt(variance)
}

func (s *AnalyticsService) GetMarketDepthAnalysis(ctx context.Context, symbol string) (map[string]interface{}, error) {
	snapshot, err := s.marketData.GetOrderBook(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}

	var bidVolume, askVolume float64
	for _, level := range snapshot.Bids {
		bidVolume += level.Quantity
	}
	for _, level := range snapshot.Asks {
		askVolume += level.Quantity
	}

	return map[string]interface{}{
		"bid_volume":      bidVolume,
		"ask_volume":      askVolume,
		"bid_ask_ratio":   bidVolume / askVolume,
		"spread":          snapshot.Asks[0].Price - snapshot.Bids[0].Price,
		"spread_percent":  (snapshot.Asks[0].Price - snapshot.Bids[0].Price) / snapshot.Bids[0].Price * 100,
		"timestamp":       snapshot.Timestamp,
		"num_bids":       len(snapshot.Bids),
		"num_asks":       len(snapshot.Asks),
	}, nil
} 