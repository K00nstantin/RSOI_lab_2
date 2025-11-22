package main

import (
	"RSOI_lab_2/pkg/database"
	"RSOI_lab_2/pkg/models"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	db = database.InitRatingDB()

	seedTestData()

	server := gin.Default()

	server.GET("/api/rating", getRating)
	server.PUT("/api/rating", updateRating)
	server.GET("/manage/health", healthcheck)

	log.Println("Rating service starting on port 8050")
	server.Run(":8050")
}

func getRating(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}

	var rating models.Rating
	if err := db.Where("username = ?", username).First(&rating).Error; err != nil {
		newRating := models.Rating{
			Username: username,
			Stars:    1,
		}
		db.Create(&newRating)
		c.JSON(http.StatusOK, gin.H{"stars": newRating.Stars})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stars": rating.Stars})
}

func updateRating(c *gin.Context) {
	var request struct {
		Username string `json:"username"`
		Stars    int    `json:"stars"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if request.Stars < 1 {
		request.Stars = 1
	} else if request.Stars > 100 {
		request.Stars = 100
	}

	var rating models.Rating
	result := db.Where("username = ?", request.Username).First(&rating)

	if result.Error == nil {
		rating.Stars = request.Stars
		db.Save(&rating)
	} else {
		rating = models.Rating{
			Username: request.Username,
			Stars:    request.Stars,
		}
		db.Create(&rating)
	}

	c.JSON(http.StatusOK, gin.H{"stars": rating.Stars})
}

func seedTestData() {
	testUsers := []models.Rating{
		{Username: "alice", Stars: 75},
		{Username: "bob", Stars: 45},
		{Username: "charlie", Stars: 90},
	}

	for _, user := range testUsers {
		db.FirstOrCreate(&user, models.Rating{Username: user.Username})
	}
	log.Println("Test data seeded")
}

func healthcheck(ctx *gin.Context) {
	sqlDB, err := db.DB()
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "DOWN",
			"details": "Database connection failed",
			"error":   err.Error(),
		})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "DOWN",
			"details": "Database ping failed",
			"error":   err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"details": "Service is healthy",
	})
}
