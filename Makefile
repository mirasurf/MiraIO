.PHONY: test test-integration test-unit build run clean setup-test-env

# Build the application
build:
	go build -o bin/miraio pkg/main.go

# Run the application
dev:
	GIN_MODE=debug go run pkg/main.go

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	go test -v -short ./...

# Run integration tests (requires MinIO to be running)
test-integration:
	@echo "Starting integration tests..."
	@echo "Make sure MinIO is running on localhost:9000 with credentials minio/minio123"
	MIRAIO_ENV=test go test -v -run Integration ./...

# Run integration tests with coverage
test-integration-coverage:
	MIRAIO_ENV=test go test -v -run Integration -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Setup test environment (create test bucket in MinIO)
setup-test-env:
	@echo "Setting up test environment..."
	@echo "Creating test bucket in MinIO..."
	@docker run --rm --network host \
		-e MC_HOST_local=http://minio:minio123@localhost:9000 \
		minio/mc \
		mb local/test-bucket --ignore-existing || true

# Clean up test artifacts
clean:
	rm -f miraio coverage.out coverage.html

# Run MinIO for testing (as Docker container)
start-minio:
	docker run -d --name miraio-test-minio \
		-p 9000:9000 -p 9001:9001 \
		-e "MINIO_ROOT_USER=minio" \
		-e "MINIO_ROOT_PASSWORD=minio123" \
		quay.io/minio/minio server /data --console-address ":9001"

# Stop test MinIO
stop-minio:
	docker stop miraio-test-minio || true
	docker rm miraio-test-minio || true

# Full test cycle with MinIO
test-full: stop-minio start-minio
	@echo "Waiting for MinIO to start..."
	@sleep 5
	@$(MAKE) setup-test-env
	@$(MAKE) test-integration
	@$(MAKE) stop-minio 