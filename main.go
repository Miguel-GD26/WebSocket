package main

import (
	"html"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return u.Host == r.Host
	},
}

func serveWs(room *ChatRoom, w http.ResponseWriter, r *http.Request) {
	rawUsername := r.URL.Query().Get("username")
	if rawUsername == "" {
		http.Error(w, "Se requiere el parámetro 'username'", http.StatusBadRequest)
		return
	}

	sanitizedUsername := html.EscapeString(rawUsername)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error al actualizar a websocket: %v", err)
		return
	}

	client := &Client{
		room:     room,
		conn:     conn,
		send:     make(chan []byte, 256),
		username: sanitizedUsername,
	}
	client.room.register <- client

	go client.writePump()
	go client.readPump()
}

func main() {
	room := newChatRoom()
	go room.run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(room, w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Servidor de Chat iniciado en http://localhost:" + port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Error fatal al iniciar servidor: %v", err)
	}
}
