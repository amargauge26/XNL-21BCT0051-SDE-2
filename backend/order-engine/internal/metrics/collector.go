package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// OrdersProcessed tracks the total number of orders processed
	OrdersProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_processed_total",
			Help: "The total number of processed orders",
		},
		[]string{"symbol", "type", "side"},
	)

	// OrderProcessingTime tracks order processing latency
	OrderProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_processing_duration_seconds",
			Help:    "Time taken to process orders",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
		},
		[]string{"symbol", "type"},
	)

	// TradesExecuted tracks the total number of trades executed
	TradesExecuted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trades_executed_total",
			Help: "The total number of executed trades",
		},
		[]string{"symbol"},
	)

	// OrderBookDepth tracks the current depth of order books
	OrderBookDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "order_book_depth",
			Help: "Current depth of the order book",
		},
		[]string{"symbol", "side"},
	)

	// ActiveOrders tracks the current number of active orders
	ActiveOrders = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_orders",
			Help: "Current number of active orders",
		},
		[]string{"symbol", "type"},
	)

	// OrderCancellations tracks the total number of cancelled orders
	OrderCancellations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_cancellations_total",
			Help: "The total number of cancelled orders",
		},
		[]string{"symbol", "type"},
	)

	// HTTPRequestDuration tracks HTTP request latencies
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler", "method", "status"},
	)

	// HTTPRequestsTotal tracks total number of HTTP requests
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"handler", "method", "status"},
	)
)

// RecordOrderProcessed increments the orders processed counter
func RecordOrderProcessed(symbol, orderType, side string) {
	OrdersProcessed.WithLabelValues(symbol, orderType, side).Inc()
}

// RecordOrderProcessingTime records the time taken to process an order
func RecordOrderProcessingTime(symbol, orderType string, duration float64) {
	OrderProcessingTime.WithLabelValues(symbol, orderType).Observe(duration)
}

// RecordTradeExecuted increments the trades executed counter
func RecordTradeExecuted(symbol string) {
	TradesExecuted.WithLabelValues(symbol).Inc()
}

// UpdateOrderBookDepth updates the order book depth gauge
func UpdateOrderBookDepth(symbol, side string, depth float64) {
	OrderBookDepth.WithLabelValues(symbol, side).Set(depth)
}

// UpdateActiveOrders updates the active orders gauge
func UpdateActiveOrders(symbol, orderType string, count float64) {
	ActiveOrders.WithLabelValues(symbol, orderType).Set(count)
}

// RecordOrderCancellation increments the order cancellations counter
func RecordOrderCancellation(symbol, orderType string) {
	OrderCancellations.WithLabelValues(symbol, orderType).Inc()
}

// RecordHTTPRequest records metrics for an HTTP request
func RecordHTTPRequest(handler, method, status string, duration float64) {
	HTTPRequestDuration.WithLabelValues(handler, method, status).Observe(duration)
	HTTPRequestsTotal.WithLabelValues(handler, method, status).Inc()
} 