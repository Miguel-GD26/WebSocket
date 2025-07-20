// hub.go
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
			// Primero, registra al cliente en el mapa
			h.clientsRWMutex.Lock()
			h.clients[client] = true
			h.clientsRWMutex.Unlock()

			log.Printf("Cliente %s conectado.", client.username)

			// Luego, crea el mensaje de unión
			joinMsg := Message{
				Type:           "user_join",
				Username:       client.username,
				MessageContent: "se ha unido al chat.",
				Timestamp:      time.Now(),
			}
			jsonMsg, _ := json.Marshal(joinMsg)
			// Envía el mensaje de unión al canal de broadcast para que TODOS lo reciban
			h.broadcast <- jsonMsg

		case client := <-h.unregister:
			// Solo si el cliente aún existe
			h.clientsRWMutex.Lock()
			if _, ok := h.clients[client]; ok {
				// Elimina al cliente
				delete(h.clients, client)
				close(client.send)
				log.Printf("Cliente %s desconectado.", client.username)

				// Crea el mensaje de desconexión
				leaveMsg := Message{
					Type:           "user_leave",
					Username:       client.username,
					MessageContent: "ha abandonado el chat.",
					Timestamp:      time.Now(),
				}
				jsonMsg, _ := json.Marshal(leaveMsg)
				// Envía el mensaje de desconexión al canal de broadcast
				h.broadcast <- jsonMsg
			}
			h.clientsRWMutex.Unlock()

		case message := <-h.broadcast:
			// Esta lógica se mantiene igual.
			h.clientsRWMutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Si el canal está bloqueado, cerramos la conexión
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
