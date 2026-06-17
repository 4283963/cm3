import { defineStore } from 'pinia'
import { stationApi } from '../api/station'

export const useStationStore = defineStore('station', {
  state: () => ({
    chargers: [],
    stationStatus: {
      totalMaxPower: 500,
      currentLimitPower: 500,
      gridLimitMode: false,
      gridLimitRatio: 1.0,
      currentTotalPower: 0,
      vipProtectedPower: 0,
      normalReducedPower: 0,
      activeChargers: 0,
      vipChargers: 0,
      idleChargers: 10,
      faultChargers: 0,
      totalChargingVehicles: 0,
    },
    powerHistory: [],
    allocationHistory: [],
    wsConnected: false,
    gridLimitSwitching: false,
  }),

  getters: {
    activeChargersList: (state) => state.chargers.filter(c => c.status === 'charging' || c.status === 'trickle'),
    powerUsagePercent: (state) => {
      const base = state.stationStatus.gridLimitMode
        ? state.stationStatus.currentLimitPower || state.stationStatus.totalMaxPower
        : state.stationStatus.totalMaxPower
      return base > 0 ? (state.stationStatus.currentTotalPower / base) * 100 : 0
    },
    vipChargersList: (state) =>
      state.chargers.filter(c => c.status === 'charging' || c.status === 'trickle')
        .filter(c => c.currentVehicle && c.currentVehicle.isVip),
    gridLimitCutPercent: (state) =>
      state.stationStatus.gridLimitMode ? ((1 - state.stationStatus.gridLimitRatio) * 100).toFixed(0) : 0,
  },

  actions: {
    async fetchChargers() {
      try {
        const res = await stationApi.getChargers()
        if (res.success) {
          this.chargers = res.data
        }
      } catch (e) {
        console.error('Failed to fetch chargers:', e)
      }
    },

    async fetchStationStatus() {
      try {
        const res = await stationApi.getStatus()
        if (res.success) {
          this.stationStatus = res.data
        }
      } catch (e) {
        console.error('Failed to fetch station status:', e)
      }
    },

    async fetchPowerHistory(params = {}) {
      try {
        const res = await stationApi.getPowerHistory(params)
        if (res.success) {
          this.powerHistory = res.data
        }
      } catch (e) {
        console.error('Failed to fetch power history:', e)
      }
    },

    async plugIn(data) {
      const res = await stationApi.plugIn(data)
      if (res.success) {
        await this.fetchChargers()
        await this.fetchStationStatus()
      }
      return res
    },

    async plugOut(chargerId) {
      const res = await stationApi.plugOut({ chargerId })
      if (res.success) {
        await this.fetchChargers()
        await this.fetchStationStatus()
      }
      return res
    },

    updateChargers(chargers) {
      this.chargers = chargers
      const actives = chargers.filter(c => c.status === 'charging' || c.status === 'trickle')
      this.stationStatus.activeChargers = actives.length
      this.stationStatus.vipChargers = actives.filter(c => c.currentVehicle && c.currentVehicle.isVip).length
      this.stationStatus.idleChargers = chargers.filter(c => c.status === 'idle').length
      this.stationStatus.faultChargers = chargers.filter(c => c.status === 'fault').length
      this.stationStatus.currentTotalPower = chargers.reduce((sum, c) => sum + (c.currentPower || 0), 0)
      this.stationStatus.vipProtectedPower = actives
        .filter(c => c.currentVehicle && c.currentVehicle.isVip)
        .reduce((sum, c) => sum + (c.currentPower || 0), 0)
    },

    updateStationStatus(status) {
      this.stationStatus = { ...this.stationStatus, ...status }
    },

    addAllocationEvent(event) {
      this.allocationHistory.unshift({
        time: new Date().toLocaleTimeString(),
        data: event,
      })
      if (this.allocationHistory.length > 50) {
        this.allocationHistory.pop()
      }
    },

    setWsConnected(connected) {
      this.wsConnected = connected
    },

    async setGridLimitMode(enabled, ratio = 0.5) {
      this.gridLimitSwitching = true
      try {
        const res = await stationApi.setGridLimit({ enabled, ratio })
        if (res.success) {
          if (res.data && res.data.stationStatus) {
            this.updateStationStatus(res.data.stationStatus)
          }
          await this.fetchChargers()
          await this.fetchPowerHistory()
        }
        return res
      } finally {
        this.gridLimitSwitching = false
      }
    },
  },
})
