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

# ==============================================================================
# Load Testing & Performance Targets (k6 & Synthetic Data Management)
# ==============================================================================
# Safety default: Load testing is disabled by default to prevent accidental runs in CI or production.
# To execute locally, pass: LOAD_TEST_ENABLED=true
LOAD_TEST_ENABLED ?= false
BASE_URL ?= http://localhost:8080
TENANTS ?= 50
USERS_PER_TENANT ?= 2

.PHONY: load-test-safety-check load-test-setup load-test-teardown load-test-status \
        load-test-baseline load-test-auth load-test-tenant load-test-mixed \
        load-test-rampup load-test-stress load-test-spike load-test-soak load-test

load-test-safety-check:
	@if [ "$$(echo $(APP_ENV) | tr '[:upper:]' '[:lower:]')" = "production" ] || [ "$$(echo $(ENVIRONMENT) | tr '[:upper:]' '[:lower:]')" = "production" ]; then \
		echo "ERROR: Load testing is strictly prohibited in production environment!"; exit 1; \
	fi
	@if [ "$(LOAD_TEST_ENABLED)" != "true" ] && [ "$(LOAD_TEST_ENABLED)" != "1" ]; then \
		echo "ERROR: Load testing is disabled by default for safety."; \
		echo "To run, explicitly enable it:"; \
		echo "  make load-test LOAD_TEST_ENABLED=true"; \
		exit 1; \
	fi

## load-test-setup: Seed synthetic multi-tenant test data and generate test-users.json
load-test-setup: load-test-safety-check
	@echo "Seeding synthetic test data..."
	go run cmd/loadtest/main.go -action setup -tenants $(TENANTS) -users-per-tenant $(USERS_PER_TENANT) -enable-load-test

## load-test-teardown: Safely clean up all synthetic load test records from database
load-test-teardown: load-test-safety-check
	@echo "Cleaning up synthetic test data..."
	go run cmd/loadtest/main.go -action teardown -enable-load-test

## load-test-status: Show current synthetic load test data counts
load-test-status: load-test-safety-check
	go run cmd/loadtest/main.go -action status -enable-load-test

## load-test-baseline: Run baseline health and readiness probe load test
load-test-baseline: load-test-safety-check
	LOAD_TEST_ENABLED=$(LOAD_TEST_ENABLED) BASE_URL=$(BASE_URL) k6 run load-tests/scenarios/health.js

## load-test-auth: Run tenant user authentication and Bcrypt load test
load-test-auth: load-test-safety-check
	LOAD_TEST_ENABLED=$(LOAD_TEST_ENABLED) BASE_URL=$(BASE_URL) k6 run load-tests/scenarios/authentication.js

## load-test-tenant: Run authenticated multi-tenant API load test
load-test-tenant: load-test-safety-check
	LOAD_TEST_ENABLED=$(LOAD_TEST_ENABLED) BASE_URL=$(BASE_URL) k6 run load-tests/scenarios/tenant.js

## load-test-mixed: Run realistic mixed workload multi-tenant test
load-test-mixed: load-test-safety-check
	LOAD_TEST_ENABLED=$(LOAD_TEST_ENABLED) BASE_URL=$(BASE_URL) k6 run load-tests/scenarios/mixed-workload.js

## load-test-rampup: Run gradual concurrency escalation ramp-up test
load-test-rampup: load-test-safety-check
	LOAD_TEST_ENABLED=$(LOAD_TEST_ENABLED) BASE_URL=$(BASE_URL) k6 run load-tests/scenarios/ramp-up.js

## load-test-stress: Run stress test to identify system breaking point and maximum stable RPS
load-test-stress: load-test-safety-check
	LOAD_TEST_ENABLED=$(LOAD_TEST_ENABLED) BASE_URL=$(BASE_URL) k6 run load-tests/scenarios/stress.js

## load-test-spike: Run burst spike test to evaluate recovery elasticity
load-test-spike: load-test-safety-check
	LOAD_TEST_ENABLED=$(LOAD_TEST_ENABLED) BASE_URL=$(BASE_URL) k6 run load-tests/scenarios/spike.js

## load-test-soak: Run endurance soak test to monitor connection leaks and stability
load-test-soak: load-test-safety-check
	LOAD_TEST_ENABLED=$(LOAD_TEST_ENABLED) BASE_URL=$(BASE_URL) k6 run load-tests/scenarios/soak.js

## load-test: Run standard verification suite (baseline -> auth -> tenant -> mixed)
load-test: load-test-safety-check load-test-baseline load-test-auth load-test-tenant load-test-mixed


