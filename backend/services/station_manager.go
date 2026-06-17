package services

import (
	"log"
	"supercharger-system/config"
	"supercharger-system/database"
	"supercharger-system/models"
	"time"
)

type StationManager struct {
	cfg          *config.StationConfig
	powerEngine  *PowerAllocationEngine
	wsHub        *WebSocketHub
	stateMachine *ChargerStateMachine
}

func NewStationManager(cfg *config.StationConfig, engine *PowerAllocationEngine, wsHub *WebSocketHub) *StationManager {
	parser := NewHardwareProtocolParser()
	stateMachine := NewChargerStateMachine(parser, wsHub)
	return &StationManager{
		cfg:          cfg,
		powerEngine:  engine,
		wsHub:        wsHub,
		stateMachine: stateMachine,
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
	IsVIP           bool    `json:"isVip"`
}

type PlugOutRequest struct {
	ChargerID int `json:"chargerId" binding:"required"`
}

type GridLimitRequest struct {
	Enabled bool    `json:"enabled"`
	Ratio   float64 `json:"ratio"`
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
	result := database.DB.Where("charger_id = ? AND status IN ?", req.ChargerID, []models.ChargerStatus{models.ChargerCharging, models.ChargerTrickle}).First(existingVehicle)
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
		IsVIP:           req.IsVIP,
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

	allocations, _, err := m.RunPowerAllocation()
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
				IsVIP:           req.IsVIP,
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
	database.DB.Where("charger_id = ? AND status IN ?", req.ChargerID, []models.ChargerStatus{models.ChargerCharging, models.ChargerTrickle}).Find(&vehicles)
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

	_, _, _ = m.RunPowerAllocation()
	return nil
}

func (m *StationManager) RunPowerAllocation() (map[int]*AllocationResult, AllocationSummary, error) {
	var chargingVehicles []models.Vehicle
	database.DB.Where("status IN ?", []models.ChargerStatus{models.ChargerCharging, models.ChargerTrickle}).Find(&chargingVehicles)

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
			IsVIP:                 v.IsVIP,
		})
	}

	results, summary := m.powerEngine.AllocatePower(cvs)

	now := time.Now()
	totalPower := 0.0
	vipCount := 0
	for _, v := range chargingVehicles {
		if result, ok := results[v.ChargerID]; ok {
			v.AllocatedPower = result.AllocatedPower
			totalPower += result.AllocatedPower
			if v.IsVIP {
				vipCount++
			}

			cv := ChargingVehicle{
				ChargerID:       v.ChargerID,
				CurrentSOC:      v.CurrentSOC,
				MaxAcceptPower:  v.MaxAcceptPower,
				BatteryCapacity: v.BatteryCapacity,
				TargetSOC:       v.TargetSOC,
				IsVIP:           v.IsVIP,
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

	m.saveStationStatus(totalPower, summary, vipCount)
	m.wsHub.BroadcastAllocationEvent(results)

	chargers, _ := m.GetAllChargers()
	m.wsHub.BroadcastChargersUpdate(chargers)

	return results, summary, nil
}

func (m *StationManager) saveStationStatus(totalPower float64, summary AllocationSummary, vipCount int) {
	var chargers []models.Charger
	database.DB.Find(&chargers)

	active := 0
	idle := 0
	fault := 0
	trickle := 0
	for _, c := range chargers {
		switch c.Status {
		case models.ChargerCharging:
			active++
		case models.ChargerTrickle:
			trickle++
			active++
		case models.ChargerIdle:
			idle++
		case models.ChargerFault:
			fault++
		default:
			log.Printf("[WARN] saveStationStatus: charger %d has unexpected status %q, treating as idle (NOT fault)", c.ID, c.Status)
			idle++
		}
	}
	_ = trickle

	_, _, currentLimitPower := m.powerEngine.GetGridLimitState()

	status := models.StationStatus{
		Timestamp:             time.Now(),
		TotalMaxPower:         m.cfg.TotalMaxPower,
		CurrentLimitPower:     currentLimitPower,
		GridLimitMode:         summary.GridLimitEnabled,
		GridLimitRatio:        summary.GridLimitRatio,
		CurrentTotalPower:     totalPower,
		VipProtectedPower:     summary.VipProtectedPower,
		NormalReducedPower:    summary.NormalCutPower,
		ActiveChargers:        active,
		VipChargers:           vipCount,
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
		if err := database.DB.Where("charger_id = ? AND status IN ?", chargers[i].ID,
			[]models.ChargerStatus{models.ChargerCharging, models.ChargerTrickle}).
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
	vipCount := 0
	totalPower := 0.0
	vipPower := 0.0

	for _, c := range chargers {
		switch c.Status {
		case models.ChargerCharging, models.ChargerTrickle:
			active++
			totalPower += c.CurrentPower
			if c.CurrentVehicle != nil && c.CurrentVehicle.IsVIP {
				vipCount++
				vipPower += c.CurrentPower
			}
		case models.ChargerIdle:
			idle++
		case models.ChargerFault:
			fault++
		default:
			log.Printf("[WARN] GetStationStatus: charger %d has unexpected status %q, treating as idle (NOT fault)", c.ID, c.Status)
			idle++
		}
	}

	gridLimitEnabled, gridLimitRatio, currentLimitPower := m.powerEngine.GetGridLimitState()

	status := &models.StationStatus{
		Timestamp:             time.Now(),
		TotalMaxPower:         m.cfg.TotalMaxPower,
		CurrentLimitPower:     currentLimitPower,
		GridLimitMode:         gridLimitEnabled,
		GridLimitRatio:        gridLimitRatio,
		CurrentTotalPower:     totalPower,
		VipProtectedPower:     vipPower,
		ActiveChargers:        active,
		VipChargers:           vipCount,
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
	err := database.DB.Where("charger_id = ? AND status IN ?", chargerID,
		[]models.ChargerStatus{models.ChargerCharging, models.ChargerTrickle}).
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
	_, _, _ = m.RunPowerAllocation()
	return nil
}

func (m *StationManager) SetGridLimitMode(req GridLimitRequest) (AllocationSummary, error) {
	m.powerEngine.SetGridLimit(req.Enabled, req.Ratio)

	deadline := time.After(1 * time.Second)
	done := make(chan AllocationSummary, 1)

	go func() {
		_, summary, _ := m.RunPowerAllocation()
		done <- summary
	}()

	select {
	case summary := <-done:
		log.Printf("[GRID_LIMIT] 限电模式切换: enabled=%v, ratio=%.2f, 分配完成耗时<1s",
			req.Enabled, summary.GridLimitRatio)
		return summary, nil
	case <-deadline:
		log.Printf("[WARN][GRID_LIMIT] 功率分配超时(>1s), 已异步启动")
		go func() {
			<-done
		}()
		_, gridLimitRatio, _ := m.powerEngine.GetGridLimitState()
		return AllocationSummary{
			GridLimitEnabled: req.Enabled,
			GridLimitRatio:   gridLimitRatio,
		}, nil
	}
}

func (m *StationManager) HardwareStateMachine() *ChargerStateMachine {
	return m.stateMachine
}
