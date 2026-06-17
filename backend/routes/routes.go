package routes

import (
	"supercharger-system/handlers"
	"supercharger-system/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	r *gin.Engine,
	stationHandler *handlers.StationHandler,
	wsHub *services.WebSocketHub,
) {
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	{
		station := api.Group("/station")
		{
			station.POST("/plug-in", stationHandler.PlugIn)
			station.POST("/plug-out", stationHandler.PlugOut)
			station.GET("/chargers", stationHandler.GetChargers)
			station.GET("/status", stationHandler.GetStationStatus)
			station.GET("/power-history", stationHandler.GetPowerHistory)
			station.POST("/update-soc", stationHandler.UpdateSOC)
			station.POST("/allocate", stationHandler.TriggerAllocation)
			station.POST("/hardware-frame", stationHandler.HandleHardwareFrame)
			station.POST("/grid-limit", stationHandler.SetGridLimit)
		}
	}

	r.GET("/ws", stationHandler.WebSocket)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
}
