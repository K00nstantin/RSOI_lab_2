package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/api/v1/rating", getRatingHandler)

	log.Println("Gateway service starting on port 8080")
	r.Run(":8080")
}

func getRatingHandler(c *gin.Context) {
	username := c.GetHeader("X-User-Name")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-User-Name header is required"})
		return
	}

	resp, err := http.Get("http://localhost:8050/api/rating?username=" + username)
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
