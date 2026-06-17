package services

import (
	"encoding/json"
	"log"
	"net/http"
	"supercharger-system/models"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan interface{}, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected, total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected, total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal WebSocket message: %v", err)
				h.mu.RUnlock()
				continue
			}
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WebSocketHub) BroadcastUpdate(data interface{}) {
	select {
	case h.broadcast <- data:
	default:
		log.Println("WebSocket broadcast channel full, dropping message")
	}
}

func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}
	h.register <- conn

	go func() {
		defer func() {
			h.unregister <- conn
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func (h *WebSocketHub) BroadcastChargersUpdate(chargers []models.Charger) {
	h.BroadcastUpdate(WSMessage{
		Type: "chargers_update",
		Data: chargers,
	})
}

func (h *WebSocketHub) BroadcastStationStatus(status models.StationStatus) {
	h.BroadcastUpdate(WSMessage{
		Type: "station_status",
		Data: status,
	})
}

func (h *WebSocketHub) BroadcastAllocationEvent(event interface{}) {
	h.BroadcastUpdate(WSMessage{
		Type: "allocation_event",
		Data: event,
	})
}
