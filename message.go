package main

import "time"

type Message struct {
	Type           string    `json:"type"`
	Username       string    `json:"username"`
	MessageContent string    `json:"messageContent"`
	Timestamp      time.Time `json:"timestamp"`
}
