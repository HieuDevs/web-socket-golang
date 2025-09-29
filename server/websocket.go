package main

import (
	"fmt"
	"net/http"
)

func checkOrigin(r *http.Request) bool {
	return true
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

	go m.registerClient(client)
	if !m.started {
		go m.Start()
		m.started = true
	}
	go m.readMessages(client)
	go m.heartbeat(client)
}
