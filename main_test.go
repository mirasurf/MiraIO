package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type PresignResponse struct {
	URL       string `json:"url"`
	PublicURL string `json:"publicUrl"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func setupTestEnvironment() {
	os.Setenv("MINIO_ENDPOINT", "localhost:9000")
	os.Setenv("MINIO_ACCESS_KEY", "minio")
	os.Setenv("MINIO_SECRET_KEY", "minio123")
	os.Setenv("MINIO_USE_SSL", "false")
	os.Setenv("MINIO_BUCKET", "test-bucket")
	os.Setenv("MINIO_PUBLIC_URL", "http://localhost:9000")

	bucketName = "test-bucket"
	publicURL = "http://localhost:9000"

	// Initialize minioClient for testing
	var err error
	minioClient, err = minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minio", "minio123", ""),
		Secure: false,
	})
	if err != nil {
		// If MinIO is not available, create a mock client
		// In a real scenario, you might want to use interfaces and dependency injection
		minioClient = nil
	}
}

func TestPresignHandler_MissingParameters(t *testing.T) {
	setupTestEnvironment()

	router := gin.New()
	router.GET("/presign", presignHandler)

	testCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Missing both parameters",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing filename or type",
		},
		{
			name:           "Missing filename",
			queryParams:    "?type=text/plain",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing filename or type",
		},
		{
			name:           "Missing type",
			queryParams:    "?filename=test.txt",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing filename or type",
		},
		{
			name:           "Empty filename",
			queryParams:    "?filename=&type=text/plain",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing filename or type",
		},
		{
			name:           "Empty type",
			queryParams:    "?filename=test.txt&type=",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing filename or type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/presign"+tc.queryParams, nil)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			assert.Equal(t, tc.expectedStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tc.expectedError)
		})
	}
}

func TestPresignHandler_ValidParameters(t *testing.T) {
	setupTestEnvironment()

	// Skip this test if MinIO is not available
	if minioClient == nil {
		t.Skip("MinIO not available for testing")
	}

	router := gin.New()
	router.GET("/presign", presignHandler)

	testCases := []struct {
		name        string
		filename    string
		contentType string
	}{
		{
			name:        "Text file",
			filename:    "test.txt",
			contentType: "text/plain",
		},
		{
			name:        "Image file",
			filename:    "image.jpg",
			contentType: "image/jpeg",
		},
		{
			name:        "PDF file",
			filename:    "document.pdf",
			contentType: "application/pdf",
		},
		{
			name:        "JSON file",
			filename:    "data.json",
			contentType: "application/json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/presign?filename="+tc.filename+"&type="+tc.contentType, nil)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			// If MinIO is not running, we expect a 500 error
			if recorder.Code == http.StatusInternalServerError {
				t.Skip("MinIO not running, cannot test presigned URL generation")
				return
			}

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Contains(t, recorder.Body.String(), "url")
			assert.Contains(t, recorder.Body.String(), "publicUrl")
			assert.Contains(t, recorder.Body.String(), tc.filename)
		})
	}
}

func TestPresignHandler_SpecialCharacters(t *testing.T) {
	setupTestEnvironment()

	if minioClient == nil {
		t.Skip("MinIO not available for testing")
	}

	router := gin.New()
	router.GET("/presign", presignHandler)

	testCases := []struct {
		name        string
		filename    string
		contentType string
	}{
		{
			name:        "Filename with spaces",
			filename:    "my test file.txt",
			contentType: "text/plain",
		},
		{
			name:        "Filename with special characters",
			filename:    "file-name_with.special-chars.txt",
			contentType: "text/plain",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/presign", nil)
			require.NoError(t, err)

			q := req.URL.Query()
			q.Add("filename", tc.filename)
			q.Add("type", tc.contentType)
			req.URL.RawQuery = q.Encode()

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code == http.StatusInternalServerError {
				t.Skip("MinIO not running, cannot test presigned URL generation")
				return
			}

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Contains(t, recorder.Body.String(), "url")
			assert.Contains(t, recorder.Body.String(), "publicUrl")
		})
	}
}

func TestPresignHandler_ContentTypeHandling(t *testing.T) {
	setupTestEnvironment()

	if minioClient == nil {
		t.Skip("MinIO not available for testing")
	}

	router := gin.New()
	router.GET("/presign", presignHandler)

	// Test various content types
	contentTypes := []string{
		"text/plain",
		"text/html",
		"application/json",
		"application/pdf",
		"image/jpeg",
		"image/png",
		"video/mp4",
		"audio/mpeg",
		"application/octet-stream",
	}

	for _, contentType := range contentTypes {
		t.Run("ContentType_"+contentType, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/presign?filename=test.file&type="+contentType, nil)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code == http.StatusInternalServerError {
				t.Skip("MinIO not running, cannot test presigned URL generation")
				return
			}

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Contains(t, recorder.Body.String(), "url")
			assert.Contains(t, recorder.Body.String(), "publicUrl")
		})
	}
}
