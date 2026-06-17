package services

import (
	"fmt"
	"supercharger-system/config"
	"supercharger-system/database"
	"supercharger-system/models"
	"time"
)

type PowerAllocationEngine struct {
	cfg *config.StationConfig
}

func NewPowerAllocationEngine(cfg *config.StationConfig) *PowerAllocationEngine {
	return &PowerAllocationEngine{cfg: cfg}
}

type ChargingVehicle struct {
	ChargerID             int
	VIN                   string
	CurrentSOC            float64
	MaxAcceptPower        float64
	BatteryCapacity       float64
	TargetSOC             float64
	CurrentAllocatedPower float64
}

type AllocationResult struct {
	ChargerID      int
	VIN            string
	AllocatedPower float64
	Reason         string
}

func (e *PowerAllocationEngine) AllocatePower(vehicles []ChargingVehicle) map[int]*AllocationResult {
	results := make(map[int]*AllocationResult)
	activeCount := len(vehicles)

	if activeCount == 0 {
		return results
	}

	totalRequestedPower := 0.0
	requestedPowers := make(map[int]float64)
	weights := make(map[int]float64)
	totalWeight := 0.0

	for _, v := range vehicles {
		requested := e.calculateRequestedPower(v)
		requestedPowers[v.ChargerID] = requested
		totalRequestedPower += requested

		weight := e.calculateWeight(v)
		weights[v.ChargerID] = weight
		totalWeight += weight

		results[v.ChargerID] = &AllocationResult{
			ChargerID: v.ChargerID,
			VIN:       v.VIN,
		}
	}

	if totalRequestedPower <= e.cfg.TotalMaxPower {
		for _, v := range vehicles {
			results[v.ChargerID].AllocatedPower = requestedPowers[v.ChargerID]
			results[v.ChargerID].Reason = "功率充足，按需求分配"
		}
		e.saveAllocationRecords(results, vehicles, totalRequestedPower)
		return results
	}

	for _, v := range vehicles {
		ratio := weights[v.ChargerID] / totalWeight
		allocated := e.cfg.TotalMaxPower * ratio

		if allocated > requestedPowers[v.ChargerID] {
			surplus := allocated - requestedPowers[v.ChargerID]
			allocated = requestedPowers[v.ChargerID]
			results[v.ChargerID].AllocatedPower = allocated
			results[v.ChargerID].Reason = "满足需求，多余功率重新分配"

			remainingVehicles := make([]ChargingVehicle, 0)
			for _, vv := range vehicles {
				if results[vv.ChargerID].AllocatedPower < requestedPowers[vv.ChargerID] {
					remainingVehicles = append(remainingVehicles, vv)
				}
			}
			e.redistributeSurplus(results, remainingVehicles, requestedPowers, surplus)
		} else {
			results[v.ChargerID].AllocatedPower = allocated
			results[v.ChargerID].Reason = fmt.Sprintf("按SOC权重分配(权重:%.2f)", weights[v.ChargerID])
		}
	}

	for _, v := range vehicles {
		result := results[v.ChargerID]
		if result.AllocatedPower > v.MaxAcceptPower {
			result.AllocatedPower = v.MaxAcceptPower
			result.Reason += "，限制为车辆最大接受功率"
		}
		charger := &models.Charger{}
		if err := database.DB.First(charger, v.ChargerID).Error; err == nil {
			if result.AllocatedPower > charger.MaxPower {
				result.AllocatedPower = charger.MaxPower
				result.Reason += "，限制为充电桩最大功率"
			}
		}
	}

	e.saveAllocationRecords(results, vehicles, e.cfg.TotalMaxPower)
	return results
}

func (e *PowerAllocationEngine) calculateRequestedPower(v ChargingVehicle) float64 {
	socDiff := v.TargetSOC - v.CurrentSOC
	if socDiff <= 0 {
		return 0
	}

	var power float64
	switch {
	case v.CurrentSOC < 20:
		power = v.MaxAcceptPower * 0.95
	case v.CurrentSOC < 50:
		power = v.MaxAcceptPower * 0.9
	case v.CurrentSOC < 80:
		power = v.MaxAcceptPower * 0.7
	case v.CurrentSOC < 90:
		power = v.MaxAcceptPower * 0.4
	case v.CurrentSOC < 95:
		power = v.MaxAcceptPower * 0.2
	default:
		power = v.MaxAcceptPower * 0.05
	}

	remainingEnergy := v.BatteryCapacity * (socDiff / 100)
	if power*0.5 > remainingEnergy {
		power = remainingEnergy / 0.5
	}

	if power < 0 {
		power = 0
	}

	return power
}

func (e *PowerAllocationEngine) calculateWeight(v ChargingVehicle) float64 {
	var weight float64

	if v.CurrentSOC < 20 {
		weight = 2.0
	} else if v.CurrentSOC < 50 {
		weight = 1.5
	} else if v.CurrentSOC < 80 {
		weight = 1.0
	} else if v.CurrentSOC < 90 {
		weight = 0.6
	} else {
		weight = 0.3
	}

	urgencyFactor := 1.0
	if v.CurrentSOC < 10 {
		urgencyFactor = 1.5
	} else if v.CurrentSOC < 15 {
		urgencyFactor = 1.2
	}

	weight *= urgencyFactor
	return weight
}

func (e *PowerAllocationEngine) redistributeSurplus(results map[int]*AllocationResult, remaining []ChargingVehicle, requested map[int]float64, surplus float64) {
	if len(remaining) == 0 || surplus <= 0 {
		return
	}

	totalRemainingNeed := 0.0
	for _, v := range remaining {
		need := requested[v.ChargerID] - results[v.ChargerID].AllocatedPower
		if need > 0 {
			totalRemainingNeed += need
		}
	}

	if totalRemainingNeed == 0 {
		return
	}

	for _, v := range remaining {
		need := requested[v.ChargerID] - results[v.ChargerID].AllocatedPower
		if need > 0 {
			share := surplus * (need / totalRemainingNeed)
			if share > need {
				share = need
			}
			results[v.ChargerID].AllocatedPower += share
		}
	}
}

func (e *PowerAllocationEngine) saveAllocationRecords(results map[int]*AllocationResult, vehicles []ChargingVehicle, totalPower float64) {
	now := time.Now()
	records := make([]models.PowerAllocationRecord, 0, len(vehicles))

	chargerInfo := make(map[int]models.Vehicle)
	for _, v := range vehicles {
		chargerInfo[v.ChargerID] = models.Vehicle{
			CurrentSOC:     v.CurrentSOC,
			MaxAcceptPower: v.MaxAcceptPower,
			VIN:            v.VIN,
		}
	}

	for chargerID, result := range results {
		info := chargerInfo[chargerID]
		record := models.PowerAllocationRecord{
			Timestamp:      now,
			ChargerID:      chargerID,
			VehicleVIN:     info.VIN,
			CurrentSOC:     info.CurrentSOC,
			MaxPower:       info.MaxAcceptPower,
			AllocatedPower: result.AllocatedPower,
			TotalPower:     totalPower,
			Reason:         result.Reason,
		}
		records = append(records, record)
	}

	if len(records) > 0 {
		database.DB.Create(&records)
	}
}

func CalculateEstimatedEndTime(vehicle ChargingVehicle, allocatedPower float64) *time.Time {
	if allocatedPower <= 0 {
		return nil
	}
	socDiff := vehicle.TargetSOC - vehicle.CurrentSOC
	if socDiff <= 0 {
		t := time.Now()
		return &t
	}
	energyNeeded := vehicle.BatteryCapacity * (socDiff / 100)
	hours := energyNeeded / allocatedPower
	endTime := time.Now().Add(time.Duration(hours * float64(time.Hour)))
	return &endTime
}
