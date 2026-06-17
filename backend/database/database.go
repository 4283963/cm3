package database

import (
	"log"
	"supercharger-system/config"
	"supercharger-system/models"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(cfg *config.DatabaseConfig) {
	var err error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		DB, err = gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database after %d retries: %v", maxRetries, err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	err = DB.AutoMigrate(
		&models.Vehicle{},
		&models.PowerAllocationRecord{},
		&models.Charger{},
		&models.StationStatus{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	initChargers()
	log.Println("Database initialized successfully")
}

func initChargers() {
	var count int64
	DB.Model(&models.Charger{}).Count(&count)
	if count > 0 {
		return
	}

	chargers := make([]models.Charger, 10)
	for i := 0; i < 10; i++ {
		chargers[i] = models.Charger{
			ID:       i + 1,
			Name:     getChargerName(i + 1),
			MaxPower: 120.0,
			Status:   models.ChargerIdle,
		}
	}
	for _, c := range chargers {
		if err := DB.Create(&c).Error; err != nil {
			log.Printf("Failed to create charger %d: %v", c.ID, err)
		}
	}
	log.Println("Initialized 10 chargers")
}

func getChargerName(id int) string {
	names := []string{"A01", "A02", "A03", "A04", "A05", "B01", "B02", "B03", "B04", "B05"}
	if id > 0 && id <= len(names) {
		return names[id-1]
	}
	return "C" + string(rune(id))
}
