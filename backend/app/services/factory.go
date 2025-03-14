package services

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/XNL-21bct0051/order-engine/config"
)

type ServiceFactory struct {
	config       *config.Config
	logger       *zap.Logger
	marketData   *MarketDataService
	risk         *RiskService
	notification *NotificationService
	analytics    *AnalyticsService
}

func NewServiceFactory(cfg *config.Config, logger *zap.Logger) (*ServiceFactory, error) {
	factory := &ServiceFactory{
		config: cfg,
		logger: logger,
	}

	// Initialize market data service
	marketData := NewMarketDataService(logger, cfg.MarketData.APIKey)
	factory.marketData = marketData

	// Initialize risk service
	risk := NewRiskService(logger, marketData)
	factory.risk = risk

	// Initialize notification service
	notification, err := NewNotificationService(logger, cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification service: %w", err)
	}
	factory.notification = notification

	// Initialize analytics service
	analytics := NewAnalyticsService(logger, marketData)
	factory.analytics = analytics

	return factory, nil
}

func (f *ServiceFactory) MarketData() *MarketDataService {
	return f.marketData
}

func (f *ServiceFactory) Risk() *RiskService {
	return f.risk
}

func (f *ServiceFactory) Notification() *NotificationService {
	return f.notification
}

func (f *ServiceFactory) Analytics() *AnalyticsService {
	return f.analytics
}

func (f *ServiceFactory) Close() error {
	if err := f.notification.Close(); err != nil {
		f.logger.Error("Failed to close notification service", zap.Error(err))
	}
	return nil
} 