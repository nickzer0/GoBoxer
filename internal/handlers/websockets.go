package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"golang.org/x/net/websocket"
)

func (repo *Repository) SendMessage(userID, message string) {
	conn, ok := repo.WsServer.Conns[userID]
	if !ok {
		log.Printf("No WebSocket connection found for user %s\n", userID)
		return
	}

	data := map[string]string{
		"type":    "message",
		"message": message,
		"status":  "success",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling message: %v\n", err)
		return
	}

	if _, err := conn.Write(jsonData); err != nil {
		log.Printf("Error sending message to user %s: %v\n", userID, err)
	}
}

func (repo *Repository) SendError(userID, message string) {
	conn, ok := repo.WsServer.Conns[userID]
	if !ok {
		log.Printf("No WebSocket connection found for user %s\n", userID)
		return
	}

	data := map[string]string{
		"type":    "message",
		"message": message,
		"status":  "error",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling error message: %v\n", err)
		return
	}

	if _, err := conn.Write(jsonData); err != nil {
		log.Printf("Error sending error message to user %s: %v\n", userID, err)
	}
}

func (repo *Repository) SendStatus(channel, messageType string, data map[string]string) {
	jsonData, err := json.Marshal(map[string]interface{}{
		"channel": channel,
		"type":    messageType,
		"data":    data,
	})
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v\n", err)
		return
	}

	for _, conn := range repo.WsServer.Conns {
		if _, err := conn.Write(jsonData); err != nil {
			log.Printf("Error broadcasting message: %v\n", err)
		}
	}
}

func (repo *Repository) Broadcast(channel, messageType string, data map[string]string) {
	jsonData, err := json.Marshal(map[string]interface{}{
		"channel": channel,
		"type":    messageType,
		"data":    data,
	})
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v\n", err)
		return
	}

	for _, conn := range repo.WsServer.Conns {
		if _, err := conn.Write(jsonData); err != nil {
			log.Printf("Error broadcasting message: %v\n", err)
		}
	}
}

func (repo *Repository) WebSocketHandler(ws *websocket.Conn, userID string) {
	defer func() {
		ws.Close()
		// Remove connection on closure
		delete(repo.WsServer.Conns, userID)
	}()

	for {
		var msg map[string]interface{}
		if err := websocket.JSON.Receive(ws, &msg); err != nil {
			if err != io.EOF {
				log.Printf("error receiving websocket msg: %v", err)
			}
			break
		}

		if action, ok := msg["action"]; ok && action == "subscribe" {
			// Handle subscription logic here
			repo.WsServer.Subscribe(ws, userID)
		}
	}
}

// HTTP handler that upgrades to WebSocket
func (m *Repository) ServeWs(w http.ResponseWriter, r *http.Request) {
	// Extract userID from session
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))

	// Upgrade to WebSocket connection
	websocket.Handler(func(ws *websocket.Conn) {
		m.WebSocketHandler(ws, userID) // Pass userID to the WebSocket handler
	}).ServeHTTP(w, r)
}
