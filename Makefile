.PHONY: all run build test test-race lint fmt migrate-up migrate-down docker-up docker-down docker-build logs clean

# Binary names
BIN_API = bin/api
BIN_MIGRATE = bin/migrate

all: build

## run: Run API application locally
run:
	go run cmd/api/main.go

## build: Build API and migrator binaries
build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BIN_API) ./cmd/api
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BIN_MIGRATE) ./cmd/migrate

## test: Run all unit and integration tests
test:
	go test -v ./...

## test-race: Run tests with data race detector enabled
test-race:
	go test -v -race ./...

## lint: Run go vet and static checks
lint:
	go vet ./...

## fmt: Format all Go code using gofmt
fmt:
	gofmt -s -w .

## migrate-up: Run database schema migrations up
migrate-up:
	go run cmd/migrate/main.go -direction up

## migrate-down: Rollback the latest database migration
migrate-down:
	go run cmd/migrate/main.go -direction down

## docker-up: Build and start all services via Docker Compose
docker-up:
	docker compose up --build -d

## docker-down: Stop all services and networks
docker-down:
	docker compose down

## docker-build: Rebuild container images
docker-build:
	docker compose build

## logs: View unified logs from Docker Compose
logs:
	docker compose logs -f

## clean: Clean compiled binaries
clean:
	rm -rf bin/
