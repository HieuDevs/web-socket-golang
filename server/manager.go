package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	webSocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type Client struct {
	conn   *websocket.Conn
	userId string
	room   string
}

type Message struct {
	Room    string `json:"room"`
	UserId  string `json:"userId"`
	Content []byte `json:"content"`
}

type Room struct {
	clients map[*websocket.Conn]*Client
}

type Manager struct {
	rooms      map[string]*Room
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
	started    bool
}

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

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userId := r.URL.Query().Get("userId")
	room := r.URL.Query().Get("room")

	if userId == "" || room == "" {
		http.Error(w, "userId and room parameters are required", http.StatusBadRequest)
		conn.Close()
		return
	}

	client := &Client{
		conn:   conn,
		userId: userId,
		room:   room,
	}

	fmt.Printf("WebSocket connection established - User: %s, Room: %s, Addr: %s\n", userId, room, conn.RemoteAddr())

	if !m.started {
		go m.Start()
		m.started = true
	}

	go m.readMessages(client)
	m.registerClient(client)
}

func (m *Manager) registerClient(client *Client) {
	m.register <- client
}

func (m *Manager) unregisterClient(client *Client) {
	m.unregister <- client
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

func (m *Manager) onClientUnregister(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if room, exists := m.rooms[client.room]; exists {
		if _, clientExists := room.clients[client.conn]; clientExists {
			m.removeClientFromRoom(client)
			if len(room.clients) == 0 {
				m.deleteRoom(client.room)
			}
		}
	}
}

func (m *Manager) onClientRegister(client *Client) {
	m.mutex.Lock()
	if m.rooms[client.room] == nil {
		m.rooms[client.room] = &Room{
			clients: make(map[*websocket.Conn]*Client),
		}
	}
	m.rooms[client.room].clients[client.conn] = client
	m.mutex.Unlock()

	fmt.Printf("User %s joined room %s\n", client.userId, client.room)

	welcomeMessage := fmt.Sprintf("Welcome %s to room %s!", client.userId, client.room)
	m.broadcast <- Message{
		Room:    client.room,
		UserId:  "system",
		Content: []byte(welcomeMessage),
	}
}

func (m *Manager) readMessages(client *Client) {
	defer m.unregisterClient(client)

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

func (m *Manager) removeClientFromRoom(client *Client) {
	defer m.mutex.Unlock()
	m.mutex.Lock()
	if room, exists := m.rooms[client.room]; exists {
		delete(room.clients, client.conn)
		client.conn.Close()
		fmt.Printf("User %s left room %s\n", client.userId, client.room)
	}
}

func (m *Manager) deleteRoom(room string) {
	defer m.mutex.Unlock()
	m.mutex.Lock()
	delete(m.rooms, room)
	fmt.Printf("Room %s is now empty and removed\n", room)
}
