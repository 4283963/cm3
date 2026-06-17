package main

import (
	"log"
	"supercharger-system/config"
	"supercharger-system/database"
	"supercharger-system/handlers"
	"supercharger-system/routes"
	"supercharger-system/services"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	database.InitDB(&cfg.Database)

	wsHub := services.NewWebSocketHub()
	go wsHub.Run()

	powerEngine := services.NewPowerAllocationEngine(&cfg.Station)
	stationManager := services.NewStationManager(&cfg.Station, powerEngine, wsHub)

	stationHandler := handlers.NewStationHandler(stationManager, wsHub)

	r := gin.Default()
	routes.SetupRoutes(r, stationHandler, wsHub)

	go startAllocationTicker(stationManager)

	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func startAllocationTicker(sm *services.StationManager) {
	allocateTicker := time.NewTicker(5 * time.Second)
	socSimTicker := time.NewTicker(3 * time.Second)

	defer allocateTicker.Stop()
	defer socSimTicker.Stop()

	for {
		select {
		case <-allocateTicker.C:
			_, _ = sm.RunPowerAllocation()

		case <-socSimTicker.C:
			simulateSOCConsumption(sm)
		}
	}
}

func simulateSOCConsumption(sm *services.StationManager) {
	chargers, err := sm.GetAllChargers()
	if err != nil {
		return
	}

	for _, c := range chargers {
		if c.Status != "charging" || c.CurrentVehicle == nil || c.CurrentPower <= 0 {
			continue
		}

		v := c.CurrentVehicle
		energyKWh := (c.CurrentPower * 3) / 3600
		socIncrease := (energyKWh / v.BatteryCapacity) * 100

		newSOC := v.CurrentSOC + socIncrease
		if newSOC > v.TargetSOC {
			newSOC = v.TargetSOC
		}

		if newSOC > v.CurrentSOC {
			_ = sm.UpdateVehicleSOC(c.ID, newSOC)
		}
	}
}
