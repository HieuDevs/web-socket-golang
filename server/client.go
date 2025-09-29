package main

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func (m *Manager) registerClient(client *Client) {
	m.register <- client
}

func (m *Manager) unregisterClient(client *Client) {
	m.unregister <- client
}

func (m *Manager) onClientRegister(client *Client) {
	m.mutex.Lock()
	m.addClientToRoom(client)
	m.mutex.Unlock()

	m.sendWelcomeMessage(client)
}

func (m *Manager) onClientUnregister(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	userId := client.userId
	roomName := client.room
	room, exists := m.getRoom(roomName)
	if !exists {
		return
	}

	if _, clientExists := room.clients[userId]; clientExists {
		m.removeClientFromRoom(client)
	}
}

func (m *Manager) heartbeat(client *Client) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			fmt.Printf("Error sending heartbeat to user %s: %v\n", client.userId, err)
			m.removeClientFromRoom(client)
			m.unregisterClient(client)
			return
		}
	}
}

func (m *Manager) readMessages(client *Client) {
	defer m.unregisterClient(client)

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		fmt.Printf("Pong received from user %s\n", client.userId)
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("User %s disconnected unexpectedly\n", client.userId)
				break
			}
			fmt.Printf("Error reading message from user %s: %v\n", client.userId, err)
			continue
		}

		m.broadcast <- Message{
			Room:    client.room,
			UserId:  client.userId,
			Content: message,
		}
	}
}

func (m *Manager) sendWelcomeMessage(client *Client) {
	welcomeMessage := fmt.Sprintf("Welcome %s to room %s!", client.userId, client.room)
	m.broadcast <- Message{
		Room:    client.room,
		UserId:  "system",
		Content: []byte(welcomeMessage),
	}
}
