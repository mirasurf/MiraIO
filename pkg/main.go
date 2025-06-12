package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var minioClient *minio.Client
var bucketName string
var publicURL string

func main() {
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
		log.Fatal("Error loading .env file")
	}

	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"
	bucketName = os.Getenv("MINIO_BUCKET")
	publicURL = os.Getenv("MINIO_PUBLIC_URL")

	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	router := gin.Default()
	router.GET("/presign", presignHandler)

	fmt.Println("Server running on :5000")
	log.Fatal(router.Run(":5000"))
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
