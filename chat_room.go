package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type ChatRoom struct {
	clients map[*Client]bool

	clientsMutex sync.RWMutex

	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	quit       chan bool
}

func newChatRoom() *ChatRoom {
	return &ChatRoom{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		quit:       make(chan bool),
	}
}

func (r *ChatRoom) run() {
	for {
		select {
		case client := <-r.register:
			r.clientsMutex.Lock()
			r.clients[client] = true
			r.clientsMutex.Unlock()

			log.Printf("Cliente %s conectado.", client.username)

			joinMsg := Message{Type: "user_join", Username: client.username, MessageContent: "se ha unido al chat.", Timestamp: time.Now()}
			jsonMsg, _ := json.Marshal(joinMsg)
			r.broadcast <- jsonMsg

		case client := <-r.unregister:
			r.clientsMutex.Lock()
			if _, ok := r.clients[client]; ok {
				close(client.send)
				delete(r.clients, client)

				leaveMsg := Message{Type: "user_leave", Username: client.username, MessageContent: "ha abandonado el chat.", Timestamp: time.Now()}
				jsonMsg, _ := json.Marshal(leaveMsg)
				r.broadcast <- jsonMsg
			}
			r.clientsMutex.Unlock()
			log.Printf("Cliente %s desconectado.", client.username)

		case message := <-r.broadcast:
			r.clientsMutex.RLock()
			for client := range r.clients {
				select {
				case client.send <- message:
				default:
					log.Printf("Canal de envío lleno para el cliente %s. Mensaje descartado.", client.username)
				}
			}
			r.clientsMutex.RUnlock()

		case <-r.quit:
			r.clientsMutex.Lock()
			for client := range r.clients {
				close(client.send)
			}
			r.clientsMutex.Unlock()
			return
		}
	}
}
