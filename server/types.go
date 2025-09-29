package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	maxMessageSize    = int64(2 * 1024 * 1024)
	pongWait          = 10 * time.Second
	pingInterval      = (pongWait * 9) / 10
	webSocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
)

type Client struct {
	conn   *websocket.Conn
	userId string
	room   string
}

type Message struct {
	Room    string          `json:"room"`
	UserId  string          `json:"userId"`
	Content json.RawMessage `json:"content"`
}

type Room struct {
	clients  map[string]*Client
	roomName string
}

type Manager struct {
	rooms      map[string]*Room
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
	started    bool
}
