# Go Microservices Architecture Project

A cloud-native microservices system built with Go, gRPC, and modern observability tools.

## Project Structure

```
project-root/
├── user-service/
│   ├── config/
│   │   └── config.yaml
│   ├── handler/
│   │   └── user.go
│   ├── repository/
│   │   └── user_repository.go
│   ├── tests/
│   │   └── user_service_test.go
│   ├── main.go
│   └── Dockerfile
│
├── order-service/
│   ├── config/
│   │   └── config.yaml
│   ├── handler/
│   │   └── order.go
│   ├── repository/
│   │   └── order_repository.go
│   ├── tests/
│   │   └── order_service_test.go
│   ├── main.go
│   └── Dockerfile
│
├── pkg/
│   └── genproto/
│       ├── user/
│       └── order/
│
├── monitoring/
│   ├── logger/
│   ├── metrics/
│   └── tracing/
│
├── deployments/
│   ├── kubernetes/
│   │   ├── user-service.yaml
│   │   ├── order-service.yaml
│   │   ├── mysql.yaml
│   │   ├── jaeger.yaml
│   │   ├── prometheus.yaml
│   │   └── kustomization.yaml
│   └── docker-compose.yml
│
├── go.mod
└── README.md
```

## Features

### User Service
- User management (CRUD operations)
- gRPC API with MySQL storage
- Metrics endpoint for Prometheus
- Distributed tracing with Jaeger
- Structured logging with Zap

### Order Service
- Order management and processing
- Integration with User Service
- MySQL database for persistence
- Metrics and tracing support
- Comprehensive error handling

### Infrastructure
- **Observability Stack:**
  - Metrics: Prometheus
  - Tracing: Jaeger
  - Logging: Zap
- **Database:** MySQL
- **Service Discovery:** Kubernetes DNS
- **Configuration:** YAML-based with Viper

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Kubernetes cluster (optional)
- MySQL 8.0

## Getting Started

### Local Development

1. Clone the repository:
   ```bash
   git clone https://github.com/zabilal/microservices.git
   cd microservices
   ```

2. Start the infrastructure services:
   ```bash
   docker-compose up -d mysql jaeger prometheus
   ```

3. Run the services:
   ```bash
   # Terminal 1 - User Service
   cd user-service
   go run main.go

   # Terminal 2 - Order Service
   cd order-service
   go run main.go
   ```

### Docker Deployment

Build and run all services using Docker Compose:
```bash
docker-compose up --build
```

### Kubernetes Deployment

1. Apply the Kubernetes manifests:
   ```bash
   kubectl apply -k deployments/kubernetes
   ```

2. Verify the deployment:
   ```bash
   kubectl get pods -n microservices
   ```

## Testing

### Unit Tests
```bash
# Test User Service
cd user-service
go test ./...

# Test Order Service
cd order-service
go test ./...
```

### Integration Tests
```bash
# Run integration tests
cd user-service/tests
go test -tags=integration
```

## Monitoring

- **Prometheus:** Access metrics at `http://localhost:9090`
- **Jaeger UI:** View traces at `http://localhost:16686`

## API Documentation

### User Service (gRPC)
- CreateUser
- GetUser
- UpdateUser
- DeleteUser
- ListUsers

### Order Service (gRPC)
- CreateOrder
- GetOrder
- UpdateOrder
- DeleteOrder
- ListOrders

## Configuration

Each service has its own `config.yaml` file in its respective `config` directory. Key configuration options:

- Server address and ports
- Database connection details
- Metrics endpoint
- Jaeger configuration
- Log level

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.