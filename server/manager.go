package main

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

func NewManager() *Manager {
	return &Manager{
		rooms:      make(map[string]*Room),
		broadcast:  make(chan Message, 1),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		mutex:      sync.RWMutex{},
		started:    false,
	}
}

func (m *Manager) Start() {
	for {
		select {
		case client, ok := <-m.register:
			if !ok {
				continue
			}
			m.onClientRegister(client)

		case client, ok := <-m.unregister:
			if !ok {
				continue
			}
			m.onClientUnregister(client)

		case message, ok := <-m.broadcast:
			m.onBroadcastMessage(message, ok)
		}
	}
}

func (m *Manager) onBroadcastMessage(message Message, ok bool) {

	m.mutex.RLock()
	room, exists := m.rooms[message.Room]
	m.mutex.RUnlock()

	if exists {
		for _, client := range room.clients {
			if client.userId != message.UserId {
				err := client.conn.WriteMessage(websocket.TextMessage, message.Content)
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("Error sending message to user %s: %v\n", client.userId, err)
					m.removeClientFromRoom(client)
					break
				}
				continue
			} else {
				if !ok {
					err := client.conn.WriteMessage(websocket.CloseMessage, nil)
					if err != nil {
						fmt.Printf("Error sending close message to user %s: %v\n", client.userId, err)
						m.removeClientFromRoom(client)
						break
					}
				}
			}
		}
	}
}
