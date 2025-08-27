package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func uploadFileToGCS(bucketName, objectName string, fileData []byte, contentType string) (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	wc := client.Bucket(bucketName).Object(objectName).NewWriter(ctx)
	wc.ContentType = contentType
	if _, err := wc.Write(fileData); err != nil {
		return "", err
	}
	if err := wc.Close(); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName), nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: no .env file found")
	}

	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("BUCKET_NAME is required in .env")
	}

	r := gin.Default()

	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}

		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer src.Close()

		fileBytes := make([]byte, file.Size)
		_, err = src.Read(fileBytes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		objectName := file.Filename
		publicURL, err := uploadFileToGCS(bucketName, objectName, fileBytes, file.Header.Get("Content-Type"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "upload success",
			"url":     publicURL,
		})
	})

	r.Run(":8080")
}
