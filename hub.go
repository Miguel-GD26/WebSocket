package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type Hub struct {
	clients        map[*Client]bool
	clientsRWMutex sync.RWMutex
	broadcast      chan []byte
	register       chan *Client
	unregister     chan *Client
	quit           chan bool
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		quit:       make(chan bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:

			h.clientsRWMutex.Lock()
			h.clients[client] = true
			h.clientsRWMutex.Unlock()

			log.Printf("Cliente %s conectado.", client.username)

			joinMsg := Message{
				Type:           "user_join",
				Username:       client.username,
				MessageContent: "se ha unido al chat.",
				Timestamp:      time.Now(),
			}
			jsonMsg, _ := json.Marshal(joinMsg)

			h.broadcast <- jsonMsg

		case client := <-h.unregister:

			h.clientsRWMutex.Lock()
			if _, ok := h.clients[client]; ok {

				delete(h.clients, client)
				close(client.send)
				log.Printf("Cliente %s desconectado.", client.username)

				leaveMsg := Message{
					Type:           "user_leave",
					Username:       client.username,
					MessageContent: "ha abandonado el chat.",
					Timestamp:      time.Now(),
				}
				jsonMsg, _ := json.Marshal(leaveMsg)

				h.broadcast <- jsonMsg
			}
			h.clientsRWMutex.Unlock()

		case message := <-h.broadcast:

			h.clientsRWMutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:

					close(client.send)
					delete(h.clients, client)
				}
			}
			h.clientsRWMutex.RUnlock()

		case <-h.quit:
			return
		}
	}
}
