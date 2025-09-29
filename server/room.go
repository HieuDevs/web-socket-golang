package main

import (
	"fmt"
)

func (m *Manager) createRoom(roomName string) *Room {
	room := &Room{
		clients: make(map[string]*Client),
	}
	m.rooms[roomName] = room
	fmt.Printf("Room %s created\n", roomName)
	return room
}

func (m *Manager) getRoom(roomName string) (*Room, bool) {
	room, exists := m.rooms[roomName]
	return room, exists
}

func (m *Manager) addClientToRoom(client *Client) {
	roomName := client.room
	room, exists := m.getRoom(roomName)
	if !exists {
		room = m.createRoom(roomName)
	}
	room.clients[client.userId] = client
	fmt.Printf("User %s added to room %s\n", client.userId, roomName)
}

func (m *Manager) removeClientFromRoom(client *Client) {
	roomName := client.room
	room, exists := m.getRoom(roomName)
	if !exists {
		return
	}

	delete(room.clients, client.userId)
	client.conn.Close()
	fmt.Printf("User %s removed from room %s\n", client.userId, roomName)

	if m.isRoomEmpty(roomName) {
		m.deleteRoom(roomName)
	}
}

func (m *Manager) deleteRoom(roomName string) {
	delete(m.rooms, roomName)
	fmt.Printf("Room %s deleted\n", roomName)
}

func (m *Manager) isRoomEmpty(roomName string) bool {
	room, exists := m.getRoom(roomName)
	if !exists {
		return true
	}
	return len(room.clients) == 0
}
