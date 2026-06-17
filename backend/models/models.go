package models

import (
	"time"
)

type ChargerStatus string

const (
	ChargerIdle     ChargerStatus = "idle"
	ChargerCharging ChargerStatus = "charging"
	ChargerTrickle  ChargerStatus = "trickle"
	ChargerFault    ChargerStatus = "fault"
	ChargerReserved ChargerStatus = "reserved"
)

type Vehicle struct {
	ID               uint          `gorm:"primaryKey" json:"id"`
	ChargerID        int           `gorm:"index;not null" json:"chargerId"`
	VIN              string        `gorm:"size:50;index" json:"vin"`
	LicensePlate     string        `gorm:"size:20" json:"licensePlate"`
	BatteryCapacity  float64       `json:"batteryCapacity"`
	CurrentSOC       float64       `json:"currentSoc"`
	MaxAcceptPower   float64       `json:"maxAcceptPower"`
	AllocatedPower   float64       `json:"allocatedPower"`
	TargetSOC        float64       `gorm:"default:100" json:"targetSoc"`
	StartTime        *time.Time    `json:"startTime"`
	EstimatedEndTime *time.Time    `json:"estimatedEndTime"`
	Status           ChargerStatus `gorm:"size:20;default:idle" json:"status"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
}

type PowerAllocationRecord struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Timestamp      time.Time `gorm:"index;not null" json:"timestamp"`
	ChargerID      int       `gorm:"index;not null" json:"chargerId"`
	VehicleVIN     string    `gorm:"size:50" json:"vehicleVin"`
	CurrentSOC     float64   `json:"currentSoc"`
	MaxPower       float64   `json:"maxPower"`
	AllocatedPower float64   `json:"allocatedPower"`
	TotalPower     float64   `json:"totalPower"`
	Reason         string    `gorm:"size:200" json:"reason"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Charger struct {
	ID             int           `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Name           string        `gorm:"size:50" json:"name"`
	MaxPower       float64       `json:"maxPower"`
	CurrentPower   float64       `json:"currentPower"`
	Status         ChargerStatus `gorm:"size:20;default:idle" json:"status"`
	CurrentVehicle *Vehicle      `gorm:"-" json:"currentVehicle,omitempty"`
	LastUpdate     time.Time     `json:"lastUpdate"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

type StationStatus struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	Timestamp             time.Time `gorm:"index" json:"timestamp"`
	TotalMaxPower         float64   `json:"totalMaxPower"`
	CurrentTotalPower     float64   `json:"currentTotalPower"`
	ActiveChargers        int       `json:"activeChargers"`
	IdleChargers          int       `json:"idleChargers"`
	FaultChargers         int       `json:"faultChargers"`
	TotalChargingVehicles int       `json:"totalChargingVehicles"`
	CreatedAt             time.Time `json:"createdAt"`
}

func (Vehicle) TableName() string               { return "vehicles" }
func (PowerAllocationRecord) TableName() string { return "power_allocation_records" }
func (Charger) TableName() string               { return "chargers" }
func (StationStatus) TableName() string         { return "station_status" }
