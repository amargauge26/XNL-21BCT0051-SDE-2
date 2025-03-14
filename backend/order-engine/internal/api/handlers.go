package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/XNL-21bct0051-SDE-2/order-engine/pkg/matching"
	"github.com/XNL-21bct0051-SDE-2/order-engine/pkg/types"
)

type Handler struct {
	engine *matching.MatchingEngine
	logger *zap.Logger
}

func NewHandler(engine *matching.MatchingEngine, logger *zap.Logger) *Handler {
	return &Handler{
		engine: engine,
		logger: logger,
	}
}

func RegisterRoutes(r *gin.Engine, engine *matching.MatchingEngine, logger *zap.Logger) {
	h := NewHandler(engine, logger)

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// Order endpoints
		v1.POST("/orders", h.CreateOrder)
		v1.GET("/orders/:id", h.GetOrder)
		v1.DELETE("/orders/:id", h.CancelOrder)
		v1.GET("/orders", h.ListOrders)

		// Order book endpoints
		v1.GET("/orderbook/:symbol", h.GetOrderBook)
		v1.GET("/orderbook/:symbol/depth", h.GetOrderBookDepth)
	}
}

type CreateOrderRequest struct {
	UserID     string          `json:"user_id" binding:"required"`
	Symbol     string          `json:"symbol" binding:"required"`
	Type       types.OrderType `json:"type" binding:"required"`
	Side       types.OrderSide `json:"side" binding:"required"`
	Price      float64         `json:"price"`
	Quantity   float64         `json:"quantity" binding:"required,gt=0"`
	StopPrice  float64         `json:"stop_price,omitempty"`
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate order type and price
	if req.Type == types.LimitOrder && req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit orders require a valid price"})
		return
	}

	if req.Type == types.StopOrder && req.StopPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stop orders require a valid stop price"})
		return
	}

	order := &types.Order{
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		Symbol:      req.Symbol,
		Type:        req.Type,
		Side:        req.Side,
		Price:       req.Price,
		Quantity:    req.Quantity,
		StopPrice:   req.StopPrice,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	trades, err := h.engine.ProcessOrder(order)
	if err != nil {
		h.logger.Error("Failed to process order",
			zap.Error(err),
			zap.String("order_id", order.ID),
			zap.String("user_id", order.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"order":  order,
		"trades": trades,
	})
}

func (h *Handler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order ID is required"})
		return
	}

	// Note: This is a simplified version. In a real implementation,
	// you would need to query the order from a persistent storage.
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (h *Handler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order ID is required"})
		return
	}

	if err := h.engine.CancelOrder(orderID); err != nil {
		h.logger.Error("Failed to cancel order",
			zap.Error(err),
			zap.String("order_id", orderID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}

func (h *Handler) ListOrders(c *gin.Context) {
	// Note: This is a simplified version. In a real implementation,
	// you would need to query orders from a persistent storage with pagination.
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (h *Handler) GetOrderBook(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	snapshot, err := h.engine.GetOrderBook(symbol)
	if err != nil {
		h.logger.Error("Failed to get order book",
			zap.Error(err),
			zap.String("symbol", symbol))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

func (h *Handler) GetOrderBookDepth(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	snapshot, err := h.engine.GetOrderBook(symbol)
	if err != nil {
		h.logger.Error("Failed to get order book depth",
			zap.Error(err),
			zap.String("symbol", symbol))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return only the top N levels (configurable)
	const depthLevels = 10
	bids := snapshot.Bids
	asks := snapshot.Asks

	if len(bids) > depthLevels {
		bids = bids[:depthLevels]
	}
	if len(asks) > depthLevels {
		asks = asks[:depthLevels]
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":    symbol,
		"timestamp": snapshot.Timestamp,
		"bids":      bids,
		"asks":      asks,
	})
} 