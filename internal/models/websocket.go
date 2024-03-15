package models

import (
	"sync"

	"golang.org/x/net/websocket"
)

type WebsocketServer struct {
	Conns map[string]*websocket.Conn
	lock  sync.Mutex
}

func NewWebsocketServer() *WebsocketServer {
	return &WebsocketServer{
		Conns: make(map[string]*websocket.Conn),
	}
}

// Subscribe associates a userID with a WebSocket connection.
func (ws *WebsocketServer) Subscribe(conn *websocket.Conn, userID string) {
	ws.lock.Lock()
	defer ws.lock.Unlock()
	ws.Conns[userID] = conn
}
