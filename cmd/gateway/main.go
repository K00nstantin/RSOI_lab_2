package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var ratingServiceURL string

func main() {
	ratingServiceURL = getEnv("RATING_SERVICE_URL", "http://localhost:8050")

	r := gin.Default()

	r.GET("/api/v1/rating", getRatingHandler)
	r.GET("/manage/health", healthCheck)

	log.Println("Gateway service starting on port 8080")
	r.Run(":8080")
}

func getRatingHandler(c *gin.Context) {
	username := c.GetHeader("X-User-Name")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-Name header is required"})
		return
	}

	resp, err := http.Get(ratingServiceURL + "/api/v1/rating?username=" + username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rating service unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "failed to get rating"})
		return
	}

	var ratingResponse struct {
		Stars int `json:"stars"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ratingResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stars": ratingResponse.Stars})
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"details": "Host localhost:8080 is active",
	})
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
