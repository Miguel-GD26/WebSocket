package main

import (
	"encoding/json"
	"log"
	"time"
)

// CORRECCIÓN: Se eliminó la `struct broadcastMessage` al ser innecesaria.

type ChatRoom struct {
	clients map[*Client]bool
	// CORRECCIÓN: Se eliminó `clientsMutex sync.Mutex` porque la serialización
	// a través de canales en la goroutine `run` ya previene las condiciones de carrera.

	broadcast  chan []byte // CORRECCIÓN: Canal de tipo `[]byte` para simplicidad.
	register   chan *Client
	unregister chan *Client
	quit       chan bool
}

func newChatRoom() *ChatRoom {
	return &ChatRoom{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256), // Canal de `[]byte`
		register:   make(chan *Client),
		unregister: make(chan *Client),
		quit:       make(chan bool),
	}
}

func (r *ChatRoom) run() {
	for {
		select {
		case client := <-r.register:
			// No se necesita Mutex aquí.
			r.clients[client] = true
			log.Printf("Cliente %s conectado.", client.username)

			joinMsg := Message{Type: "user_join", Username: client.username, MessageContent: "se ha unido al chat.", Timestamp: time.Now()}
			jsonMsg, _ := json.Marshal(joinMsg)
			// Se envía el mensaje de unión a todos, incluido el nuevo cliente.
			r.broadcast <- jsonMsg

		case client := <-r.unregister:
			// No se necesita Mutex aquí.
			if _, ok := r.clients[client]; ok {
				close(client.send)
				delete(r.clients, client)
				log.Printf("Cliente %s desconectado.", client.username)

				leaveMsg := Message{Type: "user_leave", Username: client.username, MessageContent: "ha abandonado el chat.", Timestamp: time.Now()}
				jsonMsg, _ := json.Marshal(leaveMsg)
				r.broadcast <- jsonMsg
			}

		case message := <-r.broadcast:
			// No se necesita Mutex aquí.
			// La lógica de broadcast es ahora mucho más simple.
			// Envía el mensaje a TODOS los clientes conectados.
			for client := range r.clients {
				select {
				case client.send <- message:
				default:
					log.Printf("Canal de envío lleno para el cliente %s. Mensaje descartado.", client.username)
					// Opcionalmente, se podría cerrar la conexión aquí.
				}
			}

		case <-r.quit:
			// No se necesita Mutex aquí.
			for client := range r.clients {
				close(client.send)
			}
			return
		}
	}
}
