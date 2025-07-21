package main

import (
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"
)

// --- Helper Functions (sin cambios) ---

func newMockClient(username string) *Client {
	return &Client{send: make(chan []byte, 256), username: username}
}

func expectMessage(t *testing.T, client *Client, expectedType string) {
	t.Helper()
	select {
	case msgBytes := <-client.send:
		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			t.Fatalf("No se pudo decodificar el mensaje: %v", err)
		}
		if msg.Type != expectedType {
			t.Errorf("Se esperaba un mensaje de tipo '%s', pero se recibió '%s'", expectedType, msg.Type)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Timeout esperando el mensaje de tipo '%s' para el cliente %s", expectedType, client.username)
	}
}

func assertChannelEmpty(t *testing.T, client *Client) {
	t.Helper()
	select {
	case msg, ok := <-client.send:
		if !ok {
			// El canal está cerrado y vacío, lo cual está bien.
			return
		}
		t.Fatalf("Se esperaba que el canal estuviera vacío, pero se recibió: %s", string(msg))
	case <-time.After(50 * time.Millisecond):
	}
}

// NUEVO HELPER para esperar a que un canal se cierre
func expectChannelClosed(t *testing.T, client *Client) {
	t.Helper()
	select {
	case _, ok := <-client.send:
		if ok {
			t.Fatalf("Se esperaba que el canal para %s estuviera cerrado, pero se recibió un mensaje.", client.username)
		}
		// Si !ok, el canal está cerrado. ¡Éxito!
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Timeout esperando que el canal para %s se cerrara.", client.username)
	}
}

// --- Tests Corregidos ---

func TestChatRoomRegister(t *testing.T) {
	room := newChatRoom()
	go room.run()
	defer func() { room.quit <- true }()

	client1 := newMockClient("user1")
	client2 := newMockClient("user2")

	room.register <- client1
	expectMessage(t, client1, "user_join")
	assertChannelEmpty(t, client1)

	room.register <- client2
	expectMessage(t, client1, "user_join")
	expectMessage(t, client2, "user_join")
	assertChannelEmpty(t, client1)
	assertChannelEmpty(t, client2)
}

func TestChatRoomUnregister(t *testing.T) {
	room := newChatRoom()
	go room.run()
	defer func() { room.quit <- true }()

	client1 := newMockClient("user1")
	client2 := newMockClient("user2")

	// Setup
	room.register <- client1
	expectMessage(t, client1, "user_join")
	room.register <- client2
	expectMessage(t, client1, "user_join")
	expectMessage(t, client2, "user_join")

	// Acción
	room.unregister <- client1

	// Sincronización
	expectMessage(t, client2, "user_leave")

	// Verificación CRUCIAL:
	// El hub cierra el canal de client1. Esperamos explícitamente a que esto ocurra.
	// Esto prueba que el hub ha procesado completamente la des-registración.
	expectChannelClosed(t, client1)

	// Verificación final
	assertChannelEmpty(t, client2)
}

func TestChatRoomBroadcastToAll(t *testing.T) {
	room := newChatRoom()
	go room.run()
	defer func() { room.quit <- true }()

	sender := newMockClient("sender")
	receiver := newMockClient("receiver")

	// Setup y sincronización
	room.register <- sender
	expectMessage(t, sender, "user_join")
	room.register <- receiver
	expectMessage(t, sender, "user_join")
	expectMessage(t, receiver, "user_join")
	assertChannelEmpty(t, sender)
	assertChannelEmpty(t, receiver)

	// Acción
	chatMsg := []byte(`{"type":"chat_message", "messageContent":"hola"}`)
	room.broadcast <- chatMsg

	// Sincronización y verificación
	expectMessage(t, sender, "chat_message")
	expectMessage(t, receiver, "chat_message")
	assertChannelEmpty(t, sender)
	assertChannelEmpty(t, receiver)
}

// El resto de la prueba de concurrencia no cambia.
func TestChatRoomConcurrency_RaceCondition(t *testing.T) {
	t.Parallel()
	room := newChatRoom()
	go room.run()
	defer func() { room.quit <- true }()

	numClients := 100
	var wg sync.WaitGroup
	wg.Add(numClients)

	for i := 0; i < numClients; i++ {
		go func(i int) {
			defer wg.Done()
			client := newMockClient("user" + strconv.Itoa(i))

			room.register <- client
			msg := []byte(`{"type":"chat_message", "messageContent":"test` + strconv.Itoa(i) + `"}`)
			room.broadcast <- msg
			room.unregister <- client
		}(i)
	}
	wg.Wait()
}
