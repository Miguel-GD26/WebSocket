package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type broadcastMessage struct {
	data   []byte
	sender *Client
}

type ChatRoom struct {
	clients        map[*Client]bool
	clientsRWMutex sync.RWMutex
	broadcast      chan *broadcastMessage
	register       chan *Client
	unregister     chan *Client
	quit           chan bool
}

func newChatRoom() *ChatRoom {
	return &ChatRoom{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *broadcastMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		quit:       make(chan bool),
	}
}

func (r *ChatRoom) run() {
	for {
		select {
		case client := <-r.register:
			r.clientsRWMutex.Lock()
			r.clients[client] = true
			r.clientsRWMutex.Unlock()
			log.Printf("Cliente %s conectado.", client.username)

			joinMsg := Message{Type: "user_join", Username: client.username, MessageContent: "se ha unido al chat.", Timestamp: time.Now()}
			jsonMsg, _ := json.Marshal(joinMsg)
			r.broadcast <- &broadcastMessage{data: jsonMsg, sender: nil}

		case client := <-r.unregister:
			// Este case ahora es solo para desconexiones limpias iniciadas por el cliente.
			r.clientsRWMutex.Lock()
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				close(client.send)
				log.Printf("Cliente %s desconectado.", client.username)

				leaveMsg := Message{Type: "user_leave", Username: client.username, MessageContent: "ha abandonado el chat.", Timestamp: time.Now()}
				jsonMsg, _ := json.Marshal(leaveMsg)
				r.broadcast <- &broadcastMessage{data: jsonMsg, sender: client}
			}
			r.clientsRWMutex.Unlock()

		case message := <-r.broadcast:
			clientsToUnregister := make(map[*Client]bool)

			r.clientsRWMutex.RLock()
			for client := range r.clients {
				select {
				case client.send <- message.data:
				default:
					// Recopilar cliente lento para su eliminación.
					clientsToUnregister[client] = true
				}
			}
			r.clientsRWMutex.RUnlock()

			// Si hay clientes lentos, eliminarlos directamente.
			// Esto se hace después de liberar el RLock para poder adquirir un Lock completo.
			if len(clientsToUnregister) > 0 {
				r.clientsRWMutex.Lock()
				for client := range clientsToUnregister {
					if _, ok := r.clients[client]; ok {
						delete(r.clients, client)
						close(client.send)
						log.Printf("Cliente lento %s desconectado forzosamente.", client.username)
						// No se difunde un mensaje de "abandono" aquí para evitar complejidad.
					}
				}
				r.clientsRWMutex.Unlock()
			}

		case <-r.quit:
			return
		}
	}
}
