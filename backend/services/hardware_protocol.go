package services

import (
	"fmt"
	"log"
	"strings"
	"supercharger-system/database"
	"supercharger-system/models"
	"time"
)

type HardwareEventType string

const (
	HwStreamCc       HardwareEventType = "stream_cc"
	HwStreamCv       HardwareEventType = "stream_cv"
	HwStreamTrickle  HardwareEventType = "stream_trickle"
	HwStreamComplete HardwareEventType = "stream_complete"
	HwStreamError    HardwareEventType = "stream_error"
	HwStreamIdle     HardwareEventType = "stream_idle"
)

type HardwareEvent struct {
	ChargerID int               `json:"chargerId"`
	EventType HardwareEventType `json:"eventType"`
	RawFrame  string            `json:"rawFrame"`
	Timestamp time.Time         `json:"timestamp"`
}

type HardwareProtocolParser struct {
	statusMap map[string]HardwareEventType
}

func NewHardwareProtocolParser() *HardwareProtocolParser {
	p := &HardwareProtocolParser{
		statusMap: make(map[string]HardwareEventType),
	}

	p.statusMap["STREAM_CC"] = HwStreamCc
	p.statusMap["stream_cc"] = HwStreamCc
	p.statusMap["STREAM_CV"] = HwStreamCv
	p.statusMap["stream_cv"] = HwStreamCv
	p.statusMap["STREAM_TRICKLE"] = HwStreamTrickle
	p.statusMap["stream_trickle"] = HwStreamTrickle
	p.statusMap["Stream_Trickle"] = HwStreamTrickle
	p.statusMap["STREAM_COMPLETE"] = HwStreamComplete
	p.statusMap["stream_complete"] = HwStreamComplete
	p.statusMap["STREAM_ERROR"] = HwStreamError
	p.statusMap["stream_error"] = HwStreamError
	p.statusMap["STREAM_IDLE"] = HwStreamIdle
	p.statusMap["stream_idle"] = HwStreamIdle

	return p
}

func (p *HardwareProtocolParser) ParseSerialFrame(chargerID int, rawFrame string) (*HardwareEvent, error) {
	frame := strings.TrimSpace(rawFrame)

	statusStr := p.extractStatusField(frame)
	if statusStr == "" {
		return nil, fmt.Errorf("invalid frame format: no status field found in '%s'", rawFrame)
	}

	eventType, ok := p.statusMap[statusStr]
	if !ok {
		eventType = p.fuzzyMatch(statusStr)
		if eventType == "" {
			log.Printf("[WARN] Unknown hardware status '%s' from charger %d, treating as unknown (not fault)", statusStr, chargerID)
			return &HardwareEvent{
				ChargerID: chargerID,
				EventType: HardwareEventType("unknown:" + statusStr),
				RawFrame:  rawFrame,
				Timestamp: time.Now(),
			}, nil
		}
	}

	event := &HardwareEvent{
		ChargerID: chargerID,
		EventType: eventType,
		RawFrame:  rawFrame,
		Timestamp: time.Now(),
	}

	return event, nil
}

func (p *HardwareProtocolParser) extractStatusField(frame string) string {
	if idx := strings.Index(frame, "状态切换："); idx != -1 {
		return strings.TrimSpace(frame[idx+len("状态切换："):])
	}
	if idx := strings.Index(frame, "STATUS:"); idx != -1 {
		return strings.TrimSpace(frame[idx+len("STATUS:"):])
	}
	if !strings.Contains(frame, " ") && !strings.Contains(frame, ":") {
		return frame
	}
	return frame
}

func (p *HardwareProtocolParser) fuzzyMatch(statusStr string) HardwareEventType {
	upper := strings.ToUpper(statusStr)
	for key, val := range p.statusMap {
		if strings.ToUpper(key) == upper {
			return val
		}
	}

	lower := strings.ToLower(statusStr)
	for key, val := range p.statusMap {
		if strings.ToLower(key) == lower {
			return val
		}
	}

	return ""
}

type ChargerStateMachine struct {
	parser *HardwareProtocolParser
	wsHub  *WebSocketHub
}

func NewChargerStateMachine(parser *HardwareProtocolParser, wsHub *WebSocketHub) *ChargerStateMachine {
	return &ChargerStateMachine{
		parser: parser,
		wsHub:  wsHub,
	}
}

func (sm *ChargerStateMachine) HandleHardwareEvent(event *HardwareEvent) error {
	charger := &models.Charger{}
	if err := database.DB.First(charger, event.ChargerID).Error; err != nil {
		return fmt.Errorf("charger %d not found: %w", event.ChargerID, err)
	}

	var vehicle models.Vehicle
	database.DB.Where("charger_id = ? AND status IN ?", event.ChargerID,
		[]models.ChargerStatus{models.ChargerCharging, models.ChargerTrickle}).
		Order("created_at desc").First(&vehicle)

	newStatus, powerAdjustment, reason := sm.resolveState(charger, &vehicle, event)

	if newStatus == charger.Status && newStatus != models.ChargerFault {
		log.Printf("[STATE] Charger %d: state unchanged (%s), event %s", event.ChargerID, newStatus, event.EventType)
		return nil
	}

	log.Printf("[STATE] Charger %d: %s -> %s (event: %s, reason: %s)",
		event.ChargerID, charger.Status, newStatus, event.EventType, reason)

	charger.Status = newStatus
	charger.LastUpdate = time.Now()

	if powerAdjustment >= 0 {
		charger.CurrentPower = powerAdjustment
	}

	database.DB.Save(charger)

	if vehicle.ID > 0 {
		switch newStatus {
		case models.ChargerTrickle:
			vehicle.Status = models.ChargerTrickle
			vehicle.AllocatedPower = charger.CurrentPower
			database.DB.Save(&vehicle)
		case models.ChargerIdle:
			vehicle.Status = models.ChargerIdle
			vehicle.AllocatedPower = 0
			database.DB.Save(&vehicle)
		case models.ChargerFault:
			vehicle.Status = models.ChargerIdle
			vehicle.AllocatedPower = 0
			database.DB.Save(&vehicle)
			log.Printf("[ALERT] Charger %d forced disconnect due to genuine fault: %s", event.ChargerID, reason)
		}
	}

	if sm.wsHub != nil {
		sm.wsHub.BroadcastChargersUpdate(nil)
	}

	return nil
}

func (sm *ChargerStateMachine) resolveState(charger *models.Charger, vehicle *models.Vehicle, event *HardwareEvent) (models.ChargerStatus, float64, string) {
	switch event.EventType {
	case HwStreamCc:
		return models.ChargerCharging, -1, "恒流充电阶段(CC)"

	case HwStreamCv:
		return models.ChargerCharging, -1, "恒压充电阶段(CV)"

	case HwStreamTrickle:
		if vehicle != nil && vehicle.CurrentSOC >= 80 {
			tricklePower := vehicle.MaxAcceptPower * 0.2
			if tricklePower > charger.MaxPower*0.3 {
				tricklePower = charger.MaxPower * 0.3
			}
			return models.ChargerTrickle, tricklePower, "涓流充电阶段(Trickle), SOC>=80%, 正常降功率"
		}
		return models.ChargerCharging, -1, "涓流标识但SOC未达80%, 保持充电"

	case HwStreamComplete:
		return models.ChargerIdle, 0, "充电完成"

	case HwStreamError:
		return models.ChargerFault, 0, "硬件报告错误"

	case HwStreamIdle:
		return models.ChargerIdle, 0, "设备空闲"

	default:
		log.Printf("[STATE] Charger %d: unrecognized event '%s', keeping current state %s (NOT forcing fault)",
			charger.ID, event.EventType, charger.Status)
		return charger.Status, -1, "未知硬件事件, 保持当前状态"
	}
}
