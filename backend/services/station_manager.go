package services

import (
	"supercharger-system/config"
	"supercharger-system/database"
	"supercharger-system/models"
	"time"
)

type StationManager struct {
	cfg         *config.StationConfig
	powerEngine *PowerAllocationEngine
	wsHub       *WebSocketHub
}

func NewStationManager(cfg *config.StationConfig, engine *PowerAllocationEngine, wsHub *WebSocketHub) *StationManager {
	return &StationManager{
		cfg:         cfg,
		powerEngine: engine,
		wsHub:       wsHub,
	}
}

type PlugInRequest struct {
	ChargerID       int     `json:"chargerId" binding:"required"`
	VIN             string  `json:"vin" binding:"required"`
	LicensePlate    string  `json:"licensePlate"`
	BatteryCapacity float64 `json:"batteryCapacity" binding:"required"`
	CurrentSOC      float64 `json:"currentSoc" binding:"required"`
	MaxAcceptPower  float64 `json:"maxAcceptPower" binding:"required"`
	TargetSOC       float64 `json:"targetSoc"`
}

type PlugOutRequest struct {
	ChargerID int `json:"chargerId" binding:"required"`
}

func (m *StationManager) PlugIn(req PlugInRequest) (*models.Vehicle, error) {
	if req.TargetSOC == 0 {
		req.TargetSOC = 100
	}

	charger := &models.Charger{}
	if err := database.DB.First(charger, req.ChargerID).Error; err != nil {
		return nil, err
	}

	existingVehicle := &models.Vehicle{}
	result := database.DB.Where("charger_id = ? AND status = ?", req.ChargerID, models.ChargerCharging).First(existingVehicle)
	if result.Error == nil {
		existingVehicle.Status = models.ChargerIdle
		existingVehicle.AllocatedPower = 0
		database.DB.Save(existingVehicle)
	}

	now := time.Now()
	vehicle := &models.Vehicle{
		ChargerID:       req.ChargerID,
		VIN:             req.VIN,
		LicensePlate:    req.LicensePlate,
		BatteryCapacity: req.BatteryCapacity,
		CurrentSOC:      req.CurrentSOC,
		MaxAcceptPower:  req.MaxAcceptPower,
		TargetSOC:       req.TargetSOC,
		StartTime:       &now,
		Status:          models.ChargerCharging,
	}

	if err := database.DB.Create(vehicle).Error; err != nil {
		return nil, err
	}

	charger.Status = models.ChargerCharging
	charger.CurrentVehicle = vehicle
	charger.LastUpdate = now
	database.DB.Save(charger)

	allocations, err := m.RunPowerAllocation()
	if err != nil {
		return vehicle, nil
	}

	for chargerID, allocation := range allocations {
		if chargerID == req.ChargerID {
			vehicle.AllocatedPower = allocation.AllocatedPower
			database.DB.Save(vehicle)

			cv := ChargingVehicle{
				ChargerID:       req.ChargerID,
				CurrentSOC:      req.CurrentSOC,
				MaxAcceptPower:  req.MaxAcceptPower,
				BatteryCapacity: req.BatteryCapacity,
				TargetSOC:       req.TargetSOC,
			}
			vehicle.EstimatedEndTime = CalculateEstimatedEndTime(cv, allocation.AllocatedPower)
			database.DB.Save(vehicle)
			break
		}
	}

	return vehicle, nil
}

func (m *StationManager) PlugOut(req PlugOutRequest) error {
	charger := &models.Charger{}
	if err := database.DB.First(charger, req.ChargerID).Error; err != nil {
		return err
	}

	var vehicles []models.Vehicle
	database.DB.Where("charger_id = ? AND status = ?", req.ChargerID, models.ChargerCharging).Find(&vehicles)
	for i := range vehicles {
		vehicles[i].Status = models.ChargerIdle
		vehicles[i].AllocatedPower = 0
		database.DB.Save(&vehicles[i])
	}

	now := time.Now()
	charger.Status = models.ChargerIdle
	charger.CurrentPower = 0
	charger.CurrentVehicle = nil
	charger.LastUpdate = now
	database.DB.Save(charger)

	_, _ = m.RunPowerAllocation()
	return nil
}

func (m *StationManager) RunPowerAllocation() (map[int]*AllocationResult, error) {
	var chargingVehicles []models.Vehicle
	database.DB.Where("status = ?", models.ChargerCharging).Find(&chargingVehicles)

	cvs := make([]ChargingVehicle, 0, len(chargingVehicles))
	for _, v := range chargingVehicles {
		cvs = append(cvs, ChargingVehicle{
			ChargerID:             v.ChargerID,
			VIN:                   v.VIN,
			CurrentSOC:            v.CurrentSOC,
			MaxAcceptPower:        v.MaxAcceptPower,
			BatteryCapacity:       v.BatteryCapacity,
			TargetSOC:             v.TargetSOC,
			CurrentAllocatedPower: v.AllocatedPower,
		})
	}

	results := m.powerEngine.AllocatePower(cvs)

	now := time.Now()
	totalPower := 0.0
	for _, v := range chargingVehicles {
		if result, ok := results[v.ChargerID]; ok {
			v.AllocatedPower = result.AllocatedPower
			totalPower += result.AllocatedPower

			cv := ChargingVehicle{
				ChargerID:       v.ChargerID,
				CurrentSOC:      v.CurrentSOC,
				MaxAcceptPower:  v.MaxAcceptPower,
				BatteryCapacity: v.BatteryCapacity,
				TargetSOC:       v.TargetSOC,
			}
			v.EstimatedEndTime = CalculateEstimatedEndTime(cv, result.AllocatedPower)
			database.DB.Save(&v)

			charger := &models.Charger{}
			if err := database.DB.First(charger, v.ChargerID).Error; err == nil {
				charger.CurrentPower = result.AllocatedPower
				charger.LastUpdate = now
				database.DB.Save(charger)
			}
		}
	}

	m.saveStationStatus(totalPower)
	m.wsHub.BroadcastAllocationEvent(results)

	chargers, _ := m.GetAllChargers()
	m.wsHub.BroadcastChargersUpdate(chargers)

	return results, nil
}

func (m *StationManager) saveStationStatus(totalPower float64) {
	var chargers []models.Charger
	database.DB.Find(&chargers)

	active := 0
	idle := 0
	fault := 0
	for _, c := range chargers {
		switch c.Status {
		case models.ChargerCharging:
			active++
		case models.ChargerIdle:
			idle++
		case models.ChargerFault:
			fault++
		}
	}

	status := models.StationStatus{
		Timestamp:             time.Now(),
		TotalMaxPower:         m.cfg.TotalMaxPower,
		CurrentTotalPower:     totalPower,
		ActiveChargers:        active,
		IdleChargers:          idle,
		FaultChargers:         fault,
		TotalChargingVehicles: active,
	}
	database.DB.Create(&status)
	m.wsHub.BroadcastStationStatus(status)
}

func (m *StationManager) GetAllChargers() ([]models.Charger, error) {
	var chargers []models.Charger
	if err := database.DB.Order("id").Find(&chargers).Error; err != nil {
		return nil, err
	}

	for i := range chargers {
		var vehicle models.Vehicle
		if err := database.DB.Where("charger_id = ? AND status = ?", chargers[i].ID, models.ChargerCharging).
			Order("created_at desc").First(&vehicle).Error; err == nil {
			chargers[i].CurrentVehicle = &vehicle
		}
	}

	return chargers, nil
}

func (m *StationManager) GetStationStatus() (*models.StationStatus, error) {
	var chargers []models.Charger
	database.DB.Find(&chargers)

	active := 0
	idle := 0
	fault := 0
	totalPower := 0.0

	for _, c := range chargers {
		switch c.Status {
		case models.ChargerCharging:
			active++
			totalPower += c.CurrentPower
		case models.ChargerIdle:
			idle++
		case models.ChargerFault:
			fault++
		}
	}

	status := &models.StationStatus{
		Timestamp:             time.Now(),
		TotalMaxPower:         m.cfg.TotalMaxPower,
		CurrentTotalPower:     totalPower,
		ActiveChargers:        active,
		IdleChargers:          idle,
		FaultChargers:         fault,
		TotalChargingVehicles: active,
	}

	return status, nil
}

func (m *StationManager) GetPowerHistory(chargerID int, hours int) ([]models.PowerAllocationRecord, error) {
	if hours == 0 {
		hours = 24
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	var records []models.PowerAllocationRecord
	query := database.DB.Where("timestamp >= ?", since)
	if chargerID > 0 {
		query = query.Where("charger_id = ?", chargerID)
	}
	err := query.Order("timestamp asc").Find(&records).Error
	return records, err
}

func (m *StationManager) UpdateVehicleSOC(chargerID int, newSOC float64) error {
	var vehicle models.Vehicle
	err := database.DB.Where("charger_id = ? AND status = ?", chargerID, models.ChargerCharging).
		Order("created_at desc").First(&vehicle).Error
	if err != nil {
		return err
	}

	vehicle.CurrentSOC = newSOC
	if newSOC >= vehicle.TargetSOC {
		vehicle.Status = models.ChargerIdle
		vehicle.AllocatedPower = 0

		charger := &models.Charger{}
		if err := database.DB.First(charger, chargerID).Error; err == nil {
			charger.Status = models.ChargerIdle
			charger.CurrentPower = 0
			charger.CurrentVehicle = nil
			database.DB.Save(charger)
		}
	}

	database.DB.Save(&vehicle)
	_, _ = m.RunPowerAllocation()
	return nil
}
