// main.go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // Permitir todas las conexiones
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Se requiere el parámetro 'username'", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		username: username,
	}
	client.hub.register <- client // Registra el cliente en el hub

	// Inicia las goroutines de lectura y escritura para este cliente
	go client.writePump()
	go client.readPump()
}

func main() {
	hub := newHub()
	go hub.run() // Inicia el hub en su propia goroutine

	// Servidor de archivos para el cliente web (index.html)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// Manejador para las conexiones WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Println("Servidor de Chat iniciado en http://localhost:8080")
	//err := http.ListenAndServe(":8080", nil)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Servidor iniciado en http://localhost:" + port)
	err := http.ListenAndServe(":"+port, nil)

	if err != nil {
		log.Fatalf("Error fatal al iniciar servidor: %v", err)
	}
}
