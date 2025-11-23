package main

import (
	"RSOI_lab_2/pkg/models"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	log.Println("Starting library service...")

	// Конфигурация подключения к базе данных
	host := getEnv("DB_HOST", "postgres")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "program")
	password := getEnv("DB_PASSWORD", "test")
	dbname := getEnv("DB_NAME", "library")

	// Формируем строку подключения
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, dbname, port)

	log.Printf("Connecting to database: %s@%s:%s/%s", user, host, port, dbname)

	// Подключение к базе данных с повторными попытками
	var err error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Database connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(5 * time.Second)
		}
	}

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Автоматическое создание таблиц используя модели из pkg/models
	err = db.AutoMigrate(&models.Library{}, &models.Book{}, &models.LibraryBook{})
	if err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	log.Println("Database connected successfully")

	// Проверка подключения к базе данных
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	log.Println("Database ping successful")

	seedTestData()

	// Настройка HTTP сервера с Gin
	server := gin.Default()
	server.GET("/api/v1/libraries", getLibraries)
	server.GET("/manage/health", healthCheck)

	log.Println("Library service starting on :8060")
	if err := server.Run(":8060"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getLibraries(c *gin.Context) {
	city := c.Query("city")

	var libraries []models.Library
	query := db
	if city != "" {
		query = query.Where("city = ?", city)
	}

	if err := query.Find(&libraries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, libraries)
}

func seedTestData() {
	// Заполнение тестовыми данными
	libraries := []models.Library{
		{Name: "Central Library", Address: "123 Main St", City: "Moscow"},
		{Name: "North Library", Address: "456 North Ave", City: "Moscow"},
		{Name: "South Library", Address: "789 South St", City: "St Petersburg"},
	}

	for _, lib := range libraries {
		var existing models.Library
		if err := db.Where("name = ?", lib.Name).First(&existing).Error; err != nil {
			db.Create(&lib)
		}
	}
	log.Println("Library test data seeded")
}

func healthCheck(ctx *gin.Context) {
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
		"details": "Library service is healthy",
	})
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
