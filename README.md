# Go Microservices Architecture Project

A comprehensive, production-ready microservices system built with Go, gRPC, and modern cloud-native technologies.

## Project Structure

```
project-root/
|-- user-service/
|   |-- proto/
|   |   |-- user.proto
|   |-- server/
|   |   |-- main.go
|   |   |-- server.go
|   |   |-- user_service.go
|   |-- client/
|   |   |-- client.go
|   |-- internal/
|   |   |-- models/
|   |   |-- utils/
|   |-- tests/
|   |-- Dockerfile
|   |-- go.mod
|
|-- order-service/
|   |-- proto/
|   |   |-- order.proto
|   |-- server/
|   |   |-- main.go
|   |   |-- server.go
|   |   |-- order_service.go
|   |-- client/
|   |   |-- client.go
|   |-- internal/
|   |   |-- models/
|   |   |-- utils/
|   |-- tests/
|   |-- Dockerfile
|   |-- go.mod
|
|-- api-gateway/
|   |-- main.go
|   |-- gateway.go
|   |-- config/
|   |-- Dockerfile
|   |-- go.mod
|
|-- service-registry/
|   |-- consul/
|       |-- config.json
|
|-- proto/
|   |-- shared/
|       |-- common.proto
|
|-- deployments/
|   |-- docker-compose.yml
|
|-- monitoring/
|   |-- prometheus/
|   |-- grafana/
|
|-- docs/
|-- README.md
```

## Service Overview

### User Service
- Handles user management and authentication
- gRPC-based service with PostgreSQL storage
- Located in `user-service/`

### Order Service
- Manages order processing and tracking
- Communicates with User Service via gRPC
- Located in `order-service/`

### API Gateway
- REST API gateway for external clients
- Routes requests to appropriate microservices
- Located in `api-gateway/`

### Infrastructure
- Service Discovery: Consul
- Monitoring: Prometheus & Grafana
- Tracing: Jaeger
- Database: PostgreSQL
- Cache: Redis

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- kubectl (for Kubernetes deployment)
- Make

## Local Development

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/go-test-microservices.git
cd go-test-microservices
```

### 2. Install Dependencies

```bash
make deps
```

### 3. Set Up Local Environment

```bash
# Start infrastructure services (PostgreSQL, Redis, etc.)
make infra-up

# Apply database migrations
make migrate-up
```

### 4. Run Services Locally

```bash
# Start all services
make run

# Or start individual services
make run-user-service
make run-order-service
make run-gateway
```

### 5. Run Tests

```bash
# Run all tests
make test

# Run specific tests
make test-unit
make test-integration
make test-e2e
```

## Docker Deployment

Build and run services using Docker Compose:

```bash
# Build images
docker-compose build

# Start services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f
```

## Kubernetes Deployment

### 1. Build and Push Docker Images

```bash
# Build images
make docker-build

# Push to registry
make docker-push
```

### 2. Deploy to Kubernetes

```bash
# Apply Kubernetes manifests
kubectl apply -f deployments/k8s/

# Check deployment status
kubectl get pods -n microservices

# Get service URLs
kubectl get svc -n microservices
```

## Cloud Deployment

### AWS EKS Deployment

1. Set up EKS cluster:
```bash
eksctl create cluster -f deployments/eks/cluster.yaml
```

2. Configure AWS credentials:
```bash
aws configure
```

3. Deploy services:
```bash
make deploy-aws
```

### Google Cloud GKE Deployment

1. Set up GKE cluster:
```bash
gcloud container clusters create microservices-cluster
```

2. Configure Google Cloud credentials:
```bash
gcloud auth configure-docker
```

3. Deploy services:
```bash
make deploy-gcp
```

## Monitoring and Observability

### Prometheus Metrics
- Access Prometheus: http://localhost:9090
- Available metrics:
  * Request latency
  * Error rates
  * Resource usage

### Jaeger Tracing
- Access Jaeger UI: http://localhost:16686
- Trace information:
  * Request flow
  * Service dependencies
  * Performance bottlenecks

### Grafana Dashboards
- Access Grafana: http://localhost:3000
- Available dashboards:
  * Service Overview
  * Request Metrics
  * Resource Usage

## API Documentation

### REST API (Gateway)
- Swagger UI: http://localhost:8080/swagger/
- API Documentation: [docs/api.md](docs/api.md)

### gRPC Services
- Service definitions in [api/proto](api/proto)
- Generated documentation in [docs/grpc.md](docs/grpc.md)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contact

Your Name - [@Zakariya Raji](https://twitter.com/zabilal)
Project Link: [https://github.com/zabilal/microservices](https://github.com/zabilal/microservices)