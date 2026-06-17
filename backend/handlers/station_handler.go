package handlers

import (
	"net/http"
	"strconv"
	"supercharger-system/services"

	"github.com/gin-gonic/gin"
)

type StationHandler struct {
	stationManager *services.StationManager
	wsHub          *services.WebSocketHub
}

func NewStationHandler(sm *services.StationManager, ws *services.WebSocketHub) *StationHandler {
	return &StationHandler{
		stationManager: sm,
		wsHub:          ws,
	}
}

type HardwareFrameRequest struct {
	ChargerID int    `json:"chargerId" binding:"required"`
	Frame     string `json:"frame" binding:"required"`
}

func (h *StationHandler) HandleHardwareFrame(c *gin.Context) {
	var req HardwareFrameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	parser := services.NewHardwareProtocolParser()
	event, err := parser.ParseSerialFrame(req.ChargerID, req.Frame)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "帧解析失败: " + err.Error(),
		})
		return
	}

	stateMachine := h.stationManager.HardwareStateMachine()
	if err := stateMachine.HandleHardwareEvent(event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "状态处理失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "硬件帧处理成功",
		"eventType": event.EventType,
	})
}

func (h *StationHandler) PlugIn(c *gin.Context) {
	var req services.PlugInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	vehicle, err := h.stationManager.PlugIn(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "插桩失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "车辆插桩成功",
		"data":    vehicle,
	})
}

func (h *StationHandler) PlugOut(c *gin.Context) {
	var req services.PlugOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	err := h.stationManager.PlugOut(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "拔桩失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "车辆拔桩成功",
	})
}

func (h *StationHandler) GetChargers(c *gin.Context) {
	chargers, err := h.stationManager.GetAllChargers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取充电桩状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    chargers,
	})
}

func (h *StationHandler) GetStationStatus(c *gin.Context) {
	status, err := h.stationManager.GetStationStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取电站状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

func (h *StationHandler) GetPowerHistory(c *gin.Context) {
	chargerID, _ := strconv.Atoi(c.Query("chargerId"))
	hours, _ := strconv.Atoi(c.Query("hours"))

	records, err := h.stationManager.GetPowerHistory(chargerID, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取功率历史失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    records,
	})
}

func (h *StationHandler) UpdateSOC(c *gin.Context) {
	var req struct {
		ChargerID int     `json:"chargerId" binding:"required"`
		SOC       float64 `json:"soc" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	err := h.stationManager.UpdateVehicleSOC(req.ChargerID, req.SOC)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新SOC失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "SOC更新成功",
	})
}

func (h *StationHandler) TriggerAllocation(c *gin.Context) {
	results, _, err := h.stationManager.RunPowerAllocation()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "功率分配失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "功率分配完成",
		"data":    results,
	})
}

func (h *StationHandler) SetGridLimit(c *gin.Context) {
	var req services.GridLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	summary, err := h.stationManager.SetGridLimitMode(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "限电模式切换失败: " + err.Error(),
		})
		return
	}

	status, _ := h.stationManager.GetStationStatus()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "电网限电模式已切换，1秒内完成功率调整",
		"data": gin.H{
			"gridLimitEnabled": summary.GridLimitEnabled,
			"gridLimitRatio":   summary.GridLimitRatio,
			"summary":          summary,
			"stationStatus":    status,
		},
	})
}

func (h *StationHandler) WebSocket(c *gin.Context) {
	h.wsHub.HandleWebSocket(c.Writer, c.Request)
}
