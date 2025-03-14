package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/XNL-21bct0051-SDE-2/order-engine/config"
	"github.com/XNL-21bct0051-SDE-2/order-engine/internal/api"
	"github.com/XNL-21bct0051-SDE-2/order-engine/internal/auth"
	"github.com/XNL-21bct0051-SDE-2/order-engine/internal/cache"
	"github.com/XNL-21bct0051-SDE-2/order-engine/internal/ws"
	"github.com/XNL-21bct0051-SDE-2/order-engine/pkg/matching"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	logger, err := initLogger(cfg.Log.Level)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache(
		cfg.GetRedisAddr(),
		cfg.Redis.Password,
		cfg.Redis.DB,
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to initialize Redis cache", zap.Error(err))
	}
	defer redisCache.Close()

	// Initialize JWT service
	jwtService := auth.NewJWTService("your-secret-key", "order-engine") // TODO: Move secret to config

	// Initialize WebSocket hub
	wsHub := ws.NewHub(logger)
	go wsHub.Run()

	// Create matching engine
	engine := matching.NewMatchingEngine()

	// Initialize Gin router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(api.LoggerMiddleware(logger))
	router.Use(api.MetricsMiddleware())

	// Public routes
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now(),
		})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		// Extract user info from token
		token := c.GetHeader("Authorization")
		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Upgrade HTTP connection to WebSocket
		wsHandler := ws.NewHandler(wsHub, claims.UserID)
		wsHandler.ServeWS(c.Writer, c.Request)
	})

	// Protected routes
	v1 := router.Group("/api/v1")
	v1.Use(api.AuthMiddleware(logger))
	{
		// Order endpoints
		v1.POST("/orders", api.RequireRole(auth.RoleTrader), api.CreateOrder(engine, redisCache, wsHub))
		v1.GET("/orders/:id", api.GetOrder(redisCache))
		v1.DELETE("/orders/:id", api.RequireRole(auth.RoleTrader), api.CancelOrder(engine, redisCache))
		v1.GET("/orders", api.ListOrders(redisCache))

		// Order book endpoints
		v1.GET("/orderbook/:symbol", api.GetOrderBook(engine, redisCache))
		v1.GET("/orderbook/:symbol/depth", api.GetOrderBookDepth(engine, redisCache))

		// Trade endpoints
		v1.GET("/trades/:symbol", api.GetRecentTrades(redisCache))

		// Admin endpoints
		admin := v1.Group("/admin")
		admin.Use(api.RequireRole(auth.RoleAdmin))
		{
			admin.GET("/metrics", api.GetAdminMetrics())
			admin.POST("/symbols", api.AddSymbol())
			admin.DELETE("/symbols/:symbol", api.RemoveSymbol())
		}
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server",
			zap.String("address", srv.Addr),
			zap.String("environment", cfg.Server.Environment))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exiting")
}

func initLogger(level string) (*zap.Logger, error) {
	var cfg zap.Config

	if level == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	return cfg.Build()
} 