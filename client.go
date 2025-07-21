package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Client struct {
	room     *ChatRoom
	conn     *websocket.Conn
	send     chan []byte
	username string
}

func (c *Client) readPump() {
	defer func() {
		c.room.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, rawMessage, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error de websocket inesperado: %v", err)
			}
			break
		}

		// El cliente solo envía el contenido del mensaje, el servidor añade el resto.
		var msgContent struct {
			MessageContent string `json:"messageContent"`
		}
		if err := json.Unmarshal(rawMessage, &msgContent); err != nil {
			log.Printf("error al decodificar mensaje JSON de %s: %v", c.username, err)
			continue
		}

		// Se construye el mensaje completo en el servidor
		fullMsg := Message{
			Type:           "chat_message",
			Username:       c.username,
			MessageContent: msgContent.MessageContent,
			Timestamp:      time.Now(),
		}

		processedMessage, err := json.Marshal(fullMsg)
		if err != nil {
			log.Printf("error al codificar mensaje de %s: %v", c.username, err)
			continue
		}

		// CORRECCIÓN: Se envía el mensaje como []byte directamente, sin la struct de broadcast.
		c.room.broadcast <- processedMessage
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
