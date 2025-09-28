package main

import (
	"log"
	"net/http"
)

func main() {
	manager := NewManager()

	http.HandleFunc("/ws", manager.ServeWS)
	http.Handle("/", http.FileServer(http.Dir("../frontend")))

	log.Println("Server starting on :8080")
	log.Println("Open http://localhost:8080 to test the WebSocket connection")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}
