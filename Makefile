# Archive API Makefile

.PHONY: help run build test clean setup-auth swagger docs docker

# Default target
help:
	@echo "🚀 Archive API Commands:"
	@echo ""
	@echo "  make run          - Start the API server"
	@echo "  make build        - Build the API binary"
	@echo "  make test         - Run tests"
	@echo "  make setup-auth   - Setup initial auth users"
	@echo "  make swagger      - Open Swagger documentation"
	@echo "  make docs         - Open API documentation"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker       - Build Docker image"
	@echo ""

# Start the API server
run:
	@echo "🚀 Starting Archive API server..."
	go run cmd/main.go

# Build the API binary
build:
	@echo "🔨 Building Archive API..."
	go build -o bin/archiven-api cmd/main.go
	@echo "✅ Build complete: bin/archiven-api"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test ./... -v

# Setup initial auth users
setup-auth:
	@echo "👤 Setting up initial auth users..."
	go run scripts/setup_auth.go

# Open Swagger documentation in browser
swagger:
	@echo "📖 Opening Swagger documentation..."
	@sleep 2
	@command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:8080/swagger || \
	 command -v open >/dev/null 2>&1 && open http://localhost:8080/swagger || \
	 echo "Please open http://localhost:8080/swagger in your browser"

# Open API documentation
docs:
	@echo "📚 API Documentation available at:"
	@echo "  - Swagger UI: http://localhost:8080/swagger"
	@echo "  - YAML Spec: http://localhost:8080/swagger.yaml"
	@echo "  - Health Check: http://localhost:8080/health"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -rf logs/*.log
	@echo "✅ Clean complete"

# Validate configuration
validate-config:
	@echo "🔧 Validating configuration..."
	go run tools/test_config.go

# Start MongoDB with Podman
mongo-start:
	@echo "🍃 Starting MongoDB with Podman..."
	podman run -d --name archiven-mongo \
		-p 27017:27017 \
		-v archiven-mongo-data:/data/db \
		mongo:7.0
	@sleep 3
	@echo "✅ MongoDB started on port 27017"

# Stop MongoDB
mongo-stop:
	@echo "🛑 Stopping MongoDB..."
	podman stop archiven-mongo || true
	podman rm archiven-mongo || true
	@echo "✅ MongoDB stopped"

# MongoDB logs
mongo-logs:
	@echo "📝 MongoDB logs:"
	podman logs -f archiven-mongo

# Start full stack with Podman Compose
stack-up:
	@echo "🚀 Starting full stack with Podman Compose..."
	podman-compose up -d
	@echo "✅ Stack started!"
	@echo "   API: http://localhost:8080"
	@echo "   Swagger: http://localhost:8080/swagger"
	@echo "   MongoDB: localhost:27017"

# Stop full stack
stack-down:
	@echo "🛑 Stopping full stack..."
	podman-compose down
	@echo "✅ Stack stopped"

# View stack logs
stack-logs:
	@echo "📝 Stack logs:"
	podman-compose logs -f

# Restart stack
stack-restart: stack-down stack-up

# Build and start stack
stack-build:
	@echo "🔨 Building and starting stack..."
	podman-compose up -d --build
	@echo "✅ Stack built and started!"

# Setup development environment with MongoDB
dev-setup: mongo-start
	@echo "⏳ Waiting for MongoDB to be ready..."
	@sleep 5
	@make setup-auth
	@echo "🎯 Development environment ready!"
	@echo "MongoDB is running on port 27017"
	@echo "Run 'make run' to start the API server"

# Clean development environment
dev-clean: mongo-stop clean
	@echo "🧹 Development environment cleaned"

# Full development workflow
dev: dev-setup run

# Production build
prod-build:
	@echo "🏭 Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/archiven-api cmd/main.go
	@echo "✅ Production build complete"

# Podman build
podman-build:
	@echo "🐳 Building Podman image..."
	podman build -t archiven-api:latest .
	@echo "✅ Podman image built: archiven-api:latest"

# Start with Podman
podman-run:
	@echo "🐳 Starting with Podman..."
	podman run -p 8080:8080 --name archiven-api archiven-api:latest

# Legacy Docker commands (for compatibility)
docker: podman-build
docker-run: podman-run

# Show server status
status:
	@echo "📊 Server Status:"
	@curl -s http://localhost:8080/health 2>/dev/null | jq . || echo "❌ Server not running"

# Check MongoDB status
mongo-status:
	@echo "🍃 MongoDB Status:"
	@podman ps | grep archiven-mongo || echo "❌ MongoDB not running"

# Check all containers
containers:
	@echo "🐳 Running Containers:"
	@podman ps

# Complete status check
full-status: mongo-status status containers
	@echo "✅ Status check complete"

# Generate API client (requires openapi-generator)
generate-client:
	@echo "🔧 Generating API client..."
	@command -v openapi-generator >/dev/null 2>&1 || { echo "❌ openapi-generator not installed"; exit 1; }
	openapi-generator generate -i swagger.yaml -g go -o client/
	@echo "✅ API client generated in client/"

# Lint YAML
lint-yaml:
	@echo "🔍 Linting YAML files..."
	@command -v yamllint >/dev/null 2>&1 || { echo "❌ yamllint not installed"; exit 1; }
	yamllint swagger.yaml
	@echo "✅ YAML lint complete"

# Validate OpenAPI spec
validate-openapi:
	@echo "🔍 Validating OpenAPI specification..."
	@command -v swagger >/dev/null 2>&1 || { echo "❌ swagger CLI not installed"; exit 1; }
	swagger validate swagger.yaml
	@echo "✅ OpenAPI spec is valid"

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed"

# Format code
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

# Full development setup
setup: deps setup-auth
	@echo "🎯 Development setup complete!"
	@echo "Run 'make run' to start the server"
	@echo "Then open 'make swagger' for documentation"

# Monitor logs
logs:
	@echo "📝 Watching logs..."
	tail -f logs/*.log

# Quick test endpoints
quick-test:
	@echo "🧪 Quick endpoint tests..."
	@echo "Health Check:"
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "Login Test:"
	@curl -s -X POST http://localhost:8080/auth/login \
		-H "Content-Type: application/json" \
		-d '{"username":"admin","password":"admin123"}' | jq .

# Show all routes
routes:
	@echo "🛣️  Available Routes:"
	@echo "PUBLIC:"
	@echo "  GET    /health"
	@echo "  GET    /swagger"
	@echo "  GET    /swagger.yaml"
	@echo "  POST   /auth/login"
	@echo "  POST   /auth/refresh"
	@echo ""
	@echo "PROTECTED (require JWT):"
	@echo "  GET    /api/v1/profile"
	@echo "  POST   /api/v1/logout-all"
	@echo "  POST   /api/v1/archives"
	@echo "  GET    /api/v1/archives"
	@echo "  GET    /api/v1/archives/{id}/download"
	@echo "  DELETE /api/v1/archives/{id}"
	@echo "  DELETE /api/v1/archives/{id}/permanent"
	@echo "  POST   /api/v1/archives/{id}/restore"
	@echo "  GET    /api/v1/archives/{id}/history"
	@echo "  GET    /api/v1/archives/category/{category}"
	@echo "  GET    /api/v1/archives/tags"
	@echo "  POST   /api/v1/archives/bulk"
