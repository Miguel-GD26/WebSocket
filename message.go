// message.go
package main

import "time"

// Message define la estructura de un mensaje de chat.
type Message struct {
	Type           string    `json:"type"`
	Username       string    `json:"username"`
	MessageContent string    `json:"messageContent"`
	Timestamp      time.Time `json:"timestamp"`
}
