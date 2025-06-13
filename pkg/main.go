package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/mirago/miraio/pkg/utils"
)

var minioClient *minio.Client
var bucketName string
var publicURL string

func LoadConfig() {
	env := os.Getenv("MIRAIO_ENV")
	if env == "" {
		env = "development"
	}

	envFile := ".env." + env
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		envFile = ".env"
	}

	err := godotenv.Load(envFile)
	if err != nil {
		utils.LogFatal("Error loading .env file: %v", err)
		os.Exit(1)
	}
}

func main() {
	LoadConfig()

	utils.InitLogger()

	endpoint := os.Getenv("MIRAIO_MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MIRAIO_MINIO_ACCESS_KEY")
	secretAccessKey := os.Getenv("MIRAIO_MINIO_SECRET_KEY")
	useSSL := os.Getenv("MIRAIO_MINIO_USE_SSL") == "true"
	bucketName = os.Getenv("MIRAIO_MINIO_BUCKET")
	publicURL = os.Getenv("MIRAIO_MINIO_PUBLIC_URL")

	var err error
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		utils.LogFatal("Error initializing MinIO client: %v", err)
		os.Exit(1)
	}

	router := gin.Default()
	router.GET("/presign", presignHandler)

	port := os.Getenv("MIRAIO_PORT")
	if port == "" {
		port = "9080"
	}
	utils.LogInfo("Server running on %s", port)
	utils.LogFatal("Error starting server: %v", router.Run(":"+port))
}

func presignHandler(c *gin.Context) {
	filename := c.Query("filename")
	contentType := c.Query("type")

	if filename == "" || contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing filename or type"})
		return
	}

	reqParams := make(url.Values)
	reqParams.Set("Content-Type", contentType)

	presignedURL, err := minioClient.PresignedPutObject(context.Background(), bucketName, filename, time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate presigned URL"})
		return
	}

	publicFileURL := fmt.Sprintf("%s/%s/%s", publicURL, bucketName, filename)
	c.JSON(http.StatusOK, gin.H{
		"url":       presignedURL.String(),
		"publicUrl": publicFileURL,
	})
}
