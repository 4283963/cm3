package services

import (
	"fmt"
	"supercharger-system/config"
	"supercharger-system/database"
	"supercharger-system/models"
	"time"
)

type PowerAllocationEngine struct {
	cfg            *config.StationConfig
	gridLimitMode  bool
	gridLimitRatio float64
}

func NewPowerAllocationEngine(cfg *config.StationConfig) *PowerAllocationEngine {
	return &PowerAllocationEngine{
		cfg:            cfg,
		gridLimitMode:  false,
		gridLimitRatio: 1.0,
	}
}

func (e *PowerAllocationEngine) SetGridLimit(enabled bool, ratio float64) {
	e.gridLimitMode = enabled
	if enabled {
		if ratio <= 0 || ratio > 1 {
			ratio = e.cfg.DefaultLimitRatio
		}
		e.gridLimitRatio = ratio
	} else {
		e.gridLimitRatio = 1.0
	}
}

func (e *PowerAllocationEngine) GetGridLimitState() (bool, float64, float64) {
	currentLimit := e.cfg.TotalMaxPower * e.gridLimitRatio
	return e.gridLimitMode, e.gridLimitRatio, currentLimit
}

type ChargingVehicle struct {
	ChargerID             int
	VIN                   string
	CurrentSOC            float64
	MaxAcceptPower        float64
	BatteryCapacity       float64
	TargetSOC             float64
	CurrentAllocatedPower float64
	IsVIP                 bool
}

type AllocationResult struct {
	ChargerID      int
	VIN            string
	AllocatedPower float64
	Reason         string
	IsVIP          bool
	WasCut         bool
	Protected      bool
}

type AllocationSummary struct {
	RequestedTotal    float64
	LimitedTotal      float64
	AllocatedTotal    float64
	VipProtectedPower float64
	NormalCutPower    float64
	GridLimitEnabled  bool
	GridLimitRatio    float64
	VipCount          int
	NormalCount       int
}

func (e *PowerAllocationEngine) AllocatePower(vehicles []ChargingVehicle) (map[int]*AllocationResult, AllocationSummary) {
	results := make(map[int]*AllocationResult)
	summary := AllocationSummary{}
	activeCount := len(vehicles)

	if activeCount == 0 {
		return results, summary
	}

	gridLimitEnabled, gridLimitRatio, currentLimitPower := e.GetGridLimitState()
	summary.GridLimitEnabled = gridLimitEnabled
	summary.GridLimitRatio = gridLimitRatio
	summary.LimitedTotal = currentLimitPower

	vipList := make([]ChargingVehicle, 0)
	normalList := make([]ChargingVehicle, 0)
	totalRequestedPower := 0.0
	requestedPowers := make(map[int]float64)
	weights := make(map[int]float64)
	totalWeight := 0.0
	vipTotalWeight := 0.0
	normalTotalWeight := 0.0

	for _, v := range vehicles {
		requested := e.calculateRequestedPower(v)
		requestedPowers[v.ChargerID] = requested
		totalRequestedPower += requested

		weight := e.calculateWeight(v)
		weights[v.ChargerID] = weight
		totalWeight += weight

		if v.IsVIP {
			vipList = append(vipList, v)
			vipTotalWeight += weight
			summary.VipCount++
		} else {
			normalList = append(normalList, v)
			normalTotalWeight += weight
			summary.NormalCount++
		}

		results[v.ChargerID] = &AllocationResult{
			ChargerID: v.ChargerID,
			VIN:       v.VIN,
			IsVIP:     v.IsVIP,
		}
	}

	summary.RequestedTotal = totalRequestedPower
	effectiveMaxPower := currentLimitPower

	if totalRequestedPower <= effectiveMaxPower {
		for _, v := range vehicles {
			results[v.ChargerID].AllocatedPower = requestedPowers[v.ChargerID]
			if gridLimitEnabled {
				if v.IsVIP {
					results[v.ChargerID].Reason = "限电模式，VIP优先保障，功率充足"
					results[v.ChargerID].Protected = true
				} else {
					results[v.ChargerID].Reason = "限电模式，功率充足按需分配"
				}
			} else {
				results[v.ChargerID].Reason = "功率充足，按需求分配"
			}
			summary.AllocatedTotal += results[v.ChargerID].AllocatedPower
			if v.IsVIP {
				summary.VipProtectedPower += results[v.ChargerID].AllocatedPower
			}
		}
		e.saveAllocationRecords(results, vehicles, effectiveMaxPower)
		return results, summary
	}

	vipRequestedTotal := 0.0
	for _, v := range vipList {
		vipRequestedTotal += requestedPowers[v.ChargerID]
	}
	normalRequestedTotal := totalRequestedPower - vipRequestedTotal

	remainingAfterVip := effectiveMaxPower
	vipActualAllocated := 0.0

	if gridLimitEnabled {
		vipProtectRatio := e.cfg.VipPowerProtect
		for _, v := range vipList {
			allocated := requestedPowers[v.ChargerID] * vipProtectRatio
			if allocated > requestedPowers[v.ChargerID] {
				allocated = requestedPowers[v.ChargerID]
			}
			if remainingAfterVip-allocated < 0 {
				allocated = remainingAfterVip
			}
			results[v.ChargerID].AllocatedPower = allocated
			results[v.ChargerID].Protected = true
			results[v.ChargerID].Reason = "限电模式·VIP保障：优先分配功率"
			remainingAfterVip -= allocated
			vipActualAllocated += allocated
		}

		summary.VipProtectedPower = vipActualAllocated
		normalAvailable := remainingAfterVip
		_ = normalAvailable

		if normalTotalWeight > 0 && len(normalList) > 0 {
			for _, v := range normalList {
				ratio := weights[v.ChargerID] / normalTotalWeight
				allocated := remainingAfterVip * ratio

				if allocated > requestedPowers[v.ChargerID] {
					surplus := allocated - requestedPowers[v.ChargerID]
					allocated = requestedPowers[v.ChargerID]
					results[v.ChargerID].AllocatedPower = allocated
					results[v.ChargerID].WasCut = (allocated < requestedPowers[v.ChargerID])
					results[v.ChargerID].Reason = "限电模式·普通用户：按SOC权重分配(已限电)"
					otherNormals := make([]ChargingVehicle, 0)
					for _, vv := range normalList {
						if vv.ChargerID != v.ChargerID && results[vv.ChargerID].AllocatedPower < requestedPowers[vv.ChargerID] {
							otherNormals = append(otherNormals, vv)
						}
					}
					e.redistributeSurplus(results, otherNormals, requestedPowers, surplus)
				} else {
					results[v.ChargerID].AllocatedPower = allocated
					results[v.ChargerID].WasCut = true
					cutPct := (1 - allocated/requestedPowers[v.ChargerID]) * 100
					results[v.ChargerID].Reason = fmt.Sprintf("限电模式·普通用户：被削减%.0f%%，按SOC权重分配", cutPct)
				}
			}
		}

		normalActualAllocated := 0.0
		for _, v := range normalList {
			normalActualAllocated += results[v.ChargerID].AllocatedPower
		}
		summary.NormalCutPower = normalRequestedTotal - normalActualAllocated
	} else {
		for _, v := range vehicles {
			ratio := weights[v.ChargerID] / totalWeight
			allocated := effectiveMaxPower * ratio

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

	finalTotal := 0.0
	vipFinal := 0.0
	for _, r := range results {
		finalTotal += r.AllocatedPower
		if r.IsVIP {
			vipFinal += r.AllocatedPower
		}
	}
	summary.AllocatedTotal = finalTotal
	summary.VipProtectedPower = vipFinal

	e.saveAllocationRecords(results, vehicles, effectiveMaxPower)
	return results, summary
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

	chargerInfo := make(map[int]ChargingVehicle)
	for _, v := range vehicles {
		chargerInfo[v.ChargerID] = v
	}

	for chargerID, result := range results {
		info := chargerInfo[chargerID]
		record := models.PowerAllocationRecord{
			Timestamp:      now,
			ChargerID:      chargerID,
			VehicleVIN:     info.VIN,
			IsVIP:          info.IsVIP,
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
