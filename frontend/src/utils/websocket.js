import { useStationStore } from '../stores/station'

let ws = null
let reconnectTimer = null
let reconnectAttempts = 0
const MAX_RECONNECT_ATTEMPTS = 10
const RECONNECT_INTERVAL = 3000

export function connectWebSocket() {
  const store = useStationStore()
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/ws`

  if (ws) {
    ws.close()
  }

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    console.log('WebSocket connected')
    store.setWsConnected(true)
    reconnectAttempts = 0
    if (reconnectTimer) {
      clearInterval(reconnectTimer)
      reconnectTimer = null
    }
  }

  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data)
      handleMessage(message, store)
    } catch (e) {
      console.error('Failed to parse WebSocket message:', e)
    }
  }

  ws.onerror = (error) => {
    console.error('WebSocket error:', error)
    store.setWsConnected(false)
  }

  ws.onclose = () => {
    console.log('WebSocket disconnected')
    store.setWsConnected(false)
    scheduleReconnect()
  }
}

function handleMessage(message, store) {
  switch (message.type) {
    case 'chargers_update':
      store.updateChargers(message.data)
      break
    case 'station_status':
      store.updateStationStatus(message.data)
      break
    case 'allocation_event':
      store.addAllocationEvent(message.data)
      break
  }
}

function scheduleReconnect() {
  if (reconnectTimer || reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    return
  }

  reconnectAttempts++
  console.log(`Attempting to reconnect (${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})...`)

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    connectWebSocket()
  }, RECONNECT_INTERVAL)
}

export function disconnectWebSocket() {
  if (ws) {
    ws.close()
    ws = null
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  reconnectAttempts = 0
}
