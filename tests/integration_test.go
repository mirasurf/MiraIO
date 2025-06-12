package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var minioClient *minio.Client
var bucketName string
var publicURL string

// Integration tests that require a running MinIO instance
func TestIntegrationPresignedUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test MinIO client
	testClient, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minio", "minio123", ""),
		Secure: false,
	})
	if err != nil {
		t.Skipf("MinIO not available for integration tests: %v", err)
	}

	// Ensure test bucket exists
	bucketName := "test-bucket"
	ctx := context.Background()

	exists, err := testClient.BucketExists(ctx, bucketName)
	require.NoError(t, err)

	if !exists {
		err = testClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		require.NoError(t, err)
	}

	// Set bucket policy to allow public read access
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": "*",
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::test-bucket/*"]
			}
		]
	}`
	err = testClient.SetBucketPolicy(ctx, bucketName, policy)
	require.NoError(t, err)

	// Setup environment for our application
	os.Setenv("MINIO_ENDPOINT", "localhost:9000")
	os.Setenv("MINIO_ACCESS_KEY", "minio")
	os.Setenv("MINIO_SECRET_KEY", "minio123")
	os.Setenv("MINIO_USE_SSL", "false")
	os.Setenv("MINIO_BUCKET", bucketName)
	os.Setenv("MINIO_PUBLIC_URL", "http://localhost:9000")

	// Initialize our MinIO client
	minioClient, err = minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minio", "minio123", ""),
		Secure: false,
	})
	require.NoError(t, err)

	// Set global variables
	bucketName = "test-bucket"
	publicURL = "http://localhost:9000"

	t.Run("FullUploadWorkflow", func(t *testing.T) {
		testFileName := "integration-test-file.txt"
		testContent := "This is a test file for integration testing"
		contentType := "text/plain"

		// Clean up any existing test file
		defer func() {
			testClient.RemoveObject(ctx, bucketName, testFileName, minio.RemoveObjectOptions{})
		}()

		// Step 1: Get presigned URL
		presignedURL, err := minioClient.PresignedPutObject(ctx, bucketName, testFileName, time.Minute)
		require.NoError(t, err)
		require.NotEmpty(t, presignedURL.String())

		// Step 2: Upload file using presigned URL
		req, err := http.NewRequest("PUT", presignedURL.String(), strings.NewReader(testContent))
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Step 3: Verify file was uploaded using MinIO client
		obj, err := testClient.GetObject(ctx, bucketName, testFileName, minio.GetObjectOptions{})
		require.NoError(t, err)
		defer obj.Close()

		uploadedContent, err := io.ReadAll(obj)
		require.NoError(t, err)
		assert.Equal(t, testContent, string(uploadedContent))

		// Step 4: Wait a moment for eventual consistency
		time.Sleep(100 * time.Millisecond)

		// Step 5: Verify public URL access (now should work with bucket policy)
		expectedPublicURL := "http://localhost:9000/test-bucket/" + testFileName
		publicResp, err := http.Get(expectedPublicURL)
		require.NoError(t, err)
		defer publicResp.Body.Close()

		if publicResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(publicResp.Body)
			t.Logf("Public URL response status: %d, body: %s", publicResp.StatusCode, string(body))
		}
		assert.Equal(t, http.StatusOK, publicResp.StatusCode)

		publicContent, err := io.ReadAll(publicResp.Body)
		require.NoError(t, err)
		assert.Equal(t, testContent, string(publicContent))
	})

	t.Run("PresignedURLExpiration", func(t *testing.T) {
		testFileName := "expiration-test.txt"

		// Clean up
		defer func() {
			testClient.RemoveObject(ctx, bucketName, testFileName, minio.RemoveObjectOptions{})
		}()

		// Create a presigned URL with very short expiration (1 second)
		presignedURL, err := minioClient.PresignedPutObject(ctx, bucketName, testFileName, 1*time.Second)
		require.NoError(t, err)

		// Wait for URL to expire
		time.Sleep(3 * time.Second)

		// Try to use expired URL
		req, err := http.NewRequest("PUT", presignedURL.String(), strings.NewReader("test"))
		require.NoError(t, err)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should fail with 403 Forbidden due to expired signature
		// Note: MinIO might return different error codes for expired URLs
		if resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Unexpected success response. Status: %d, Body: %s", resp.StatusCode, string(body))
		}

		// Accept either 403 Forbidden or 400 Bad Request as valid expired URL responses
		assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest,
			"Expected 403 or 400 for expired URL, got %d", resp.StatusCode)
	})

	t.Run("LargeFileUpload", func(t *testing.T) {
		testFileName := "large-file-test.bin"

		// Create 1MB test content
		testContent := bytes.Repeat([]byte("A"), 1024*1024)

		// Clean up
		defer func() {
			testClient.RemoveObject(ctx, bucketName, testFileName, minio.RemoveObjectOptions{})
		}()

		// Get presigned URL
		presignedURL, err := minioClient.PresignedPutObject(ctx, bucketName, testFileName, 5*time.Minute)
		require.NoError(t, err)

		// Upload large file
		req, err := http.NewRequest("PUT", presignedURL.String(), bytes.NewReader(testContent))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/octet-stream")

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify file size
		objInfo, err := testClient.StatObject(ctx, bucketName, testFileName, minio.StatObjectOptions{})
		require.NoError(t, err)
		assert.Equal(t, int64(1024*1024), objInfo.Size)
	})

	t.Run("ContentTypePreservation", func(t *testing.T) {
		testCases := []struct {
			fileName    string
			contentType string
		}{
			{"test.jpg", "image/jpeg"},
			{"test.png", "image/png"},
			{"test.pdf", "application/pdf"},
			{"test.json", "application/json"},
		}

		for _, tc := range testCases {
			t.Run(tc.fileName, func(t *testing.T) {
				testContent := "test content for " + tc.fileName

				// Clean up
				defer func() {
					testClient.RemoveObject(ctx, bucketName, tc.fileName, minio.RemoveObjectOptions{})
				}()

				// Create presigned URL
				presignedURL, err := minioClient.PresignedPutObject(ctx, bucketName, tc.fileName, time.Minute)
				require.NoError(t, err)

				// Upload with specified content type
				req, err := http.NewRequest("PUT", presignedURL.String(), strings.NewReader(testContent))
				require.NoError(t, err)
				req.Header.Set("Content-Type", tc.contentType)

				client := &http.Client{Timeout: 30 * time.Second}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				// Verify content type was preserved
				objInfo, err := testClient.StatObject(ctx, bucketName, tc.fileName, minio.StatObjectOptions{})
				require.NoError(t, err)
				assert.Equal(t, tc.contentType, objInfo.ContentType)
			})
		}
	})
}

// Additional test for the API endpoint itself
func TestPresignHandler(t *testing.T) {
	// Mock environment setup
	os.Setenv("MINIO_ENDPOINT", "localhost:9000")
	os.Setenv("MINIO_ACCESS_KEY", "minio")
	os.Setenv("MINIO_SECRET_KEY", "minio123")
	os.Setenv("MINIO_USE_SSL", "false")
	os.Setenv("MINIO_BUCKET", "test-bucket")
	os.Setenv("MINIO_PUBLIC_URL", "http://localhost:9000")

	// Initialize minioClient
	var err error
	minioClient, err = minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minio", "minio123", ""),
		Secure: false,
	})
	if err != nil {
		t.Skipf("MinIO not available for handler tests: %v", err)
	}

	bucketName = "test-bucket"
	publicURL = "http://localhost:9000"

	t.Run("ValidRequest", func(t *testing.T) {
		// This would require setting up a test HTTP server
		// For now, we're testing the integration with MinIO directly
		t.Skip("Handler unit tests require HTTP server setup")
	})

	t.Run("MissingParameters", func(t *testing.T) {
		// This would test the handler with missing filename or type parameters
		t.Skip("Handler unit tests require HTTP server setup")
	})
}
