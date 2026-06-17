import { defineStore } from 'pinia'
import { stationApi } from '../api/station'

export const useStationStore = defineStore('station', {
  state: () => ({
    chargers: [],
    stationStatus: {
      totalMaxPower: 500,
      currentTotalPower: 0,
      activeChargers: 0,
      idleChargers: 10,
      faultChargers: 0,
      totalChargingVehicles: 0,
    },
    powerHistory: [],
    allocationHistory: [],
    wsConnected: false,
  }),

  getters: {
    activeChargersList: (state) => state.chargers.filter(c => c.status === 'charging' || c.status === 'trickle'),
    powerUsagePercent: (state) =>
      state.stationStatus.totalMaxPower > 0
        ? (state.stationStatus.currentTotalPower / state.stationStatus.totalMaxPower) * 100
        : 0,
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
      this.stationStatus.activeChargers = chargers.filter(c => c.status === 'charging' || c.status === 'trickle').length
      this.stationStatus.idleChargers = chargers.filter(c => c.status === 'idle').length
      this.stationStatus.faultChargers = chargers.filter(c => c.status === 'fault').length
      this.stationStatus.currentTotalPower = chargers.reduce((sum, c) => sum + (c.currentPower || 0), 0)
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
  },
})
