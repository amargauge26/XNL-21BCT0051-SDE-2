# Order Matching Engine Service

A high-performance, real-time order matching engine built with Go, designed to handle high-frequency trading operations with robust risk management and analytics capabilities.

## 🚀 Features

- **Real-time Order Matching**: Fast and efficient price-time priority matching algorithm
- **Risk Management**: Pre-trade risk checks and position monitoring
- **Market Data Integration**: Real-time price feeds and order book management
- **Analytics**: VWAP, OHLCV, Volume Profile, and Market Depth analysis
- **Real-time Notifications**: WebSocket-based updates for trades and order status
- **Role-based Access Control**: Secure API access with JWT authentication
- **Monitoring**: Prometheus metrics and Grafana dashboards

## 🛠️ Tech Stack

- **Language**: Go 1.21+
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **Message Broker**: NATS with JetStream
- **Monitoring**: Prometheus & Grafana
- **Container**: Docker & Docker Compose

## 🏃‍♂️ Quick Start

### Prerequisites

1. Docker and Docker Compose
2. Go 1.21 or higher
3. Make

### One-Line Setup (Recommended)

```bash
make docker-up && make deps && make run
```

### Step-by-Step Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/XNL-21bct0051/order-engine.git
   cd order-engine/backend
   ```

2. **Start Infrastructure Services**
   ```bash
   make docker-up
   ```

3. **Install Dependencies**
   ```bash
   make deps
   make tidy
   ```

4. **Run the Service**
   ```bash
   make run
   ```

### 🔍 Verify Installation

1. **Check Health Endpoint**
   ```bash
   curl http://localhost:8080/health
   ```

2. **Access Monitoring**
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000 (admin/admin)

## 🔧 Configuration

Edit `config/config.yaml` to customize:
- Server settings
- Database connections
- Redis cache
- NATS messaging
- JWT authentication
- Market data integration

## 📚 API Documentation

### Authentication
```bash
# Get JWT Token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "trader", "password": "password"}'
```

### Order Management
```bash
# Place New Order
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BTC-USD",
    "side": "BUY",
    "type": "LIMIT",
    "quantity": 1.0,
    "price": 50000.0
  }'
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint
```

## 🔐 Security Features

- JWT-based authentication
- Role-based access control (Admin, Trader, Analyst)
- Rate limiting
- Input validation
- Secure WebSocket connections

## 📊 Monitoring & Metrics

Key metrics available in Prometheus/Grafana:
- Order processing latency
- Matching engine throughput
- Trade volume
- System resource usage
- Error rates

## 🏗️ Project Structure

```
backend/
├── app/
│   ├── services/         # Core services
│   └── api/             # API handlers
├── pkg/
│   ├── matching/        # Matching engine
│   └── types/          # Common types
├── cmd/
│   └── server/         # Entry point
├── config/             # Configuration
└── docker-compose.yml  # Infrastructure
```

## 🛑 Common Issues & Solutions

1. **Connection Refused**
   ```bash
   # Check if services are running
   docker-compose ps
   ```

2. **Permission Denied**
   ```bash
   # Fix file permissions
   sudo chown -R $USER:$USER .
   ```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit changes
4. Push to the branch
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.


