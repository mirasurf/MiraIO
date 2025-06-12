# MiraIO - File Upload Service

MiraIO is a lightweight file upload service built with Go and Gin that provides presigned URLs for secure file uploads to MinIO/S3-compatible storage.

## Features

- **Presigned URL Generation**: Generate secure, time-limited URLs for file uploads
- **MinIO Integration**: Full compatibility with MinIO and S3-compatible storage
- **RESTful API**: Simple HTTP API built with Gin framework
- **Environment Configuration**: Flexible configuration via environment variables
- **Comprehensive Testing**: Unit and integration tests with high coverage

## API Endpoints

### GET /presign

Generate a presigned URL for file upload.

**Query Parameters:**
- `filename` (required): Name of the file to upload
- `type` (required): MIME type of the file

**Response:**
```json
{
  "url": "http://localhost:9000/bucket/file.jpg?X-Amz-Algorithm=...",
  "publicUrl": "http://localhost:9000/bucket/file.jpg"
}
```

**Example:**
```bash
curl "http://localhost:5000/presign?filename=image.jpg&type=image/jpeg"
```

## Environment Variables

Create a `.env` file or set these environment variables:

```env
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minio
MINIO_SECRET_KEY=minio123
MINIO_USE_SSL=false
MINIO_BUCKET=uploads
MINIO_PUBLIC_URL=http://localhost:9000
```

## Running the Service

### Prerequisites

- Go 1.23+
- MinIO server running (see docker-compose.yml in root)

### Development

```bash
# Install dependencies
go mod tidy

# Run the service
make run
# or
go run main.go
```

The service will start on port 5000.

## Testing

### Prerequisites for Integration Tests

Integration tests require a running MinIO instance. You can use the provided Docker setup:

```bash
# Start MinIO (from project root)
docker-compose up minio

# Or start a standalone MinIO for testing
make start-minio
```

### Running Tests

```bash
# Run all tests (unit + integration)
make test

# Run only unit tests (fast, no external dependencies)
make test-unit

# Run only integration tests (requires MinIO)
make test-integration

# Run integration tests with coverage report
make test-integration-coverage

# Full test cycle (starts/stops MinIO automatically)
make test-full
```

### Test Categories

1. **Unit Tests** (`main_test.go`):
   - HTTP endpoint testing with mocked responses
   - Input validation
   - Error handling
   - Response format verification

2. **Integration Tests** (`integration_test.go`):
   - Full MinIO integration
   - Actual file upload workflows
   - Presigned URL expiration testing
   - Large file upload testing
   - Content type preservation

### Test Environment Setup

The tests automatically:
- Set up test environment variables
- Create test buckets in MinIO
- Clean up test files after each test
- Skip integration tests if MinIO is unavailable

## Usage Example

1. **Get a presigned URL:**
```bash
curl "http://localhost:5000/presign?filename=my-image.jpg&type=image/jpeg"
```

Response:
```json
{
  "url": "http://localhost:9000/uploads/my-image.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&...",
  "publicUrl": "http://localhost:9000/uploads/my-image.jpg"
}
```

2. **Upload file using the presigned URL:**
```bash
curl -X PUT \
  -H "Content-Type: image/jpeg" \
  --data-binary @my-image.jpg \
  "http://localhost:9000/uploads/my-image.jpg?X-Amz-Algorithm=..."
```

3. **Access the uploaded file:**
```bash
curl "http://localhost:9000/uploads/my-image.jpg"
```

## Build and Deploy

```bash
# Build binary
make build

# Clean build artifacts
make clean
```

## Architecture

- **Framework**: Gin (HTTP router)
- **Storage**: MinIO/S3-compatible storage
- **Configuration**: Environment variables with .env support
- **Testing**: Testify framework with comprehensive coverage

## Security Considerations

- Presigned URLs expire after 1 minute by default
- CORS should be configured in the storage layer
- Consider implementing rate limiting for production use
- Validate file types and sizes as needed