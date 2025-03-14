package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/XNL-21bct0051-SDE-2/order-engine/pkg/types"
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	symbols  map[string]bool
	mu       sync.RWMutex
	userID   string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	logger     *zap.Logger
}

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.logger.Info("New client connected",
				zap.String("user_id", client.userID))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.logger.Info("Client disconnected",
					zap.String("user_id", client.userID))
			}

		case message := <-h.broadcast:
			var update struct {
				Symbol string `json:"symbol"`
			}
			if err := json.Unmarshal(message, &update); err != nil {
				h.logger.Error("Failed to unmarshal update", zap.Error(err))
				continue
			}

			for client := range h.clients {
				client.mu.RLock()
				subscribed := client.symbols[update.Symbol]
				client.mu.RUnlock()

				if subscribed {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}

func (c *Client) Subscribe(symbol string) {
	c.mu.Lock()
	c.symbols[symbol] = true
	c.mu.Unlock()
	c.hub.logger.Info("Client subscribed to symbol",
		zap.String("user_id", c.userID),
		zap.String("symbol", symbol))
}

func (c *Client) Unsubscribe(symbol string) {
	c.mu.Lock()
	delete(c.symbols, symbol)
	c.mu.Unlock()
	c.hub.logger.Info("Client unsubscribed from symbol",
		zap.String("user_id", c.userID),
		zap.String("symbol", symbol))
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512) // Max message size
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Error("WebSocket read error",
					zap.Error(err),
					zap.String("user_id", c.userID))
			}
			break
		}

		var cmd struct {
			Action string `json:"action"`
			Symbol string `json:"symbol"`
		}

		if err := json.Unmarshal(message, &cmd); err != nil {
			c.hub.logger.Error("Failed to unmarshal command",
				zap.Error(err),
				zap.String("user_id", c.userID))
			continue
		}

		switch cmd.Action {
		case "subscribe":
			c.Subscribe(cmd.Symbol)
		case "unsubscribe":
			c.Unsubscribe(cmd.Symbol)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastOrderBookUpdate sends order book updates to subscribed clients
func (h *Hub) BroadcastOrderBookUpdate(symbol string, snapshot *types.OrderBookSnapshot) {
	data, err := json.Marshal(struct {
		Type      string                   `json:"type"`
		Symbol    string                   `json:"symbol"`
		Timestamp time.Time                `json:"timestamp"`
		Bids      []types.OrderBookLevel   `json:"bids"`
		Asks      []types.OrderBookLevel   `json:"asks"`
	}{
		Type:      "orderbook",
		Symbol:    symbol,
		Timestamp: snapshot.Timestamp,
		Bids:      snapshot.Bids,
		Asks:      snapshot.Asks,
	})

	if err != nil {
		h.logger.Error("Failed to marshal order book update",
			zap.Error(err),
			zap.String("symbol", symbol))
		return
	}

	h.broadcast <- data
}

// BroadcastTrade sends trade updates to subscribed clients
func (h *Hub) BroadcastTrade(trade *types.Trade) {
	data, err := json.Marshal(struct {
		Type      string    `json:"type"`
		Symbol    string    `json:"symbol"`
		Price     float64   `json:"price"`
		Quantity  float64   `json:"quantity"`
		Timestamp time.Time `json:"timestamp"`
	}{
		Type:      "trade",
		Symbol:    trade.Symbol,
		Price:     trade.Price,
		Quantity:  trade.Quantity,
		Timestamp: trade.ExecutedAt,
	})

	if err != nil {
		h.logger.Error("Failed to marshal trade update",
			zap.Error(err),
			zap.String("symbol", trade.Symbol))
		return
	}

	h.broadcast <- data
} 