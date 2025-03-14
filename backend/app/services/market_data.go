package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type MarketDataService struct {
	logger     *zap.Logger
	apiKey     string
	baseURL    string
	httpClient *http.Client
	cache      sync.Map
}

type MarketPrice struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

func NewMarketDataService(logger *zap.Logger, apiKey string) *MarketDataService {
	return &MarketDataService{
		logger:  logger,
		apiKey:  apiKey,
		baseURL: "https://api.example.com/v1", // Replace with actual market data API
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *MarketDataService) GetPrice(ctx context.Context, symbol string) (*MarketPrice, error) {
	// Check cache first
	if cached, ok := s.cache.Load(symbol); ok {
		price := cached.(*MarketPrice)
		if time.Since(price.Timestamp) < time.Second*5 {
			return price, nil
		}
	}

	// Make API request
	url := fmt.Sprintf("%s/prices/%s", s.baseURL, symbol)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var price MarketPrice
	if err := json.NewDecoder(resp.Body).Decode(&price); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Update cache
	price.Timestamp = time.Now()
	s.cache.Store(symbol, &price)

	return &price, nil
}

func (s *MarketDataService) SubscribeToPrice(symbol string, updates chan<- *MarketPrice) error {
	// TODO: Implement WebSocket connection to market data provider
	return fmt.Errorf("not implemented")
}

func (s *MarketDataService) GetHistoricalPrices(ctx context.Context, symbol string, start, end time.Time) ([]*MarketPrice, error) {
	// TODO: Implement historical price data retrieval
	return nil, fmt.Errorf("not implemented")
} 