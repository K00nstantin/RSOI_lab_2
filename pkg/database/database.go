package database

import (
	"RSOI_lab_2/pkg/models"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitRatingDB() *gorm.DB {
	dsn := "host=localhost user=program password=test dbname=ratings port=5433 sslmode=disable TimeZone=UTC"
	return initDB(dsn, &models.Rating{})
}

func InitLibraryDB() *gorm.DB {
	dsn := "host=localhost user=program password=test dbname=libraries port=5433 sslmode=disable TimeZone=UTC"
	db := initDB(dsn, &models.Library{}, &models.Book{}, &models.LibraryBook{})

	return db
}

func InitReservationDB() *gorm.DB {
	dsn := "host=localhost user=program password=test dbname=reservations port=5433 sslmode=disable TimeZone=UTC"
	return initDB(dsn, &models.Reservation{})
}

func initDB(dsn string, models ...interface{}) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	err = db.AutoMigrate(models...)
	if err != nil {
		log.Fatal("Database migration failed:", err)
	}

	return db
}
