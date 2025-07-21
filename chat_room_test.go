package main

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func newMockClient(username string) *Client {
	return &Client{send: make(chan []byte, 256), username: username}
}

func drainChannel(ch chan []byte) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func TestChatRoomRegisterAndUnregister(t *testing.T) {
	room := newChatRoom()
	go room.run()
	defer func() { room.quit <- true }()
	client := newMockClient("testuser")
	room.register <- client
	time.Sleep(50 * time.Millisecond)
	room.clientsRWMutex.RLock()
	if len(room.clients) != 1 {
		t.Fatal("El cliente no se registró")
	}
	room.clientsRWMutex.RUnlock()
	room.unregister <- client
	time.Sleep(50 * time.Millisecond)
	room.clientsRWMutex.RLock()
	if len(room.clients) != 0 {
		t.Fatal("El cliente no se des-registró")
	}
	room.clientsRWMutex.RUnlock()
}

func TestChatRoomBroadcast(t *testing.T) {
	room := newChatRoom()
	go room.run()
	defer func() { room.quit <- true }()
	client1 := newMockClient("user1")
	client2 := newMockClient("user2")
	room.register <- client1
	room.register <- client2
	time.Sleep(50 * time.Millisecond)

	drainChannel(client1.send)
	drainChannel(client2.send)

	message := []byte("hola a todos")
	room.broadcast <- &broadcastMessage{data: message, sender: nil}

	timeout := time.After(100 * time.Millisecond)
	select {
	case received1 := <-client1.send:
		if string(received1) != string(message) {
			t.Errorf("Client1 recibió '%s', se esperaba '%s'", string(received1), string(message))
		}
	case <-timeout:
		t.Fatal("Client1 no recibió el mensaje a tiempo")
	}
	select {
	case received2 := <-client2.send:
		if string(received2) != string(message) {
			t.Errorf("Client2 recibió '%s', se esperaba '%s'", string(received2), string(message))
		}
	case <-timeout:
		t.Fatal("Client2 no recibió el mensaje a tiempo")
	}
}

func TestChatRoomBroadcastWithSlowClient(t *testing.T) {
	room := newChatRoom()
	go room.run()
	defer func() { room.quit <- true }()

	normalClient := newMockClient("normalUser")
	slowClient := newMockClient("slowUser")
	slowClient.send = make(chan []byte, 1)

	room.register <- normalClient
	room.register <- slowClient
	time.Sleep(50 * time.Millisecond)
	drainChannel(normalClient.send)
	drainChannel(slowClient.send)

	slowClient.send <- []byte("blocker")

	triggerMsg := []byte("trigger message")
	room.broadcast <- &broadcastMessage{data: triggerMsg, sender: nil}

	success := false
	for i := 0; i < 20; i++ {
		room.clientsRWMutex.RLock()
		count := len(room.clients)
		room.clientsRWMutex.RUnlock()
		if count == 1 {
			success = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !success {
		t.Fatal("La prueba falló: el cliente lento no fue desconectado a tiempo.")
	}

	select {
	case msg := <-normalClient.send:
		if string(msg) != string(triggerMsg) {
			t.Errorf("Se esperaba el mensaje de trigger, se obtuvo: %s", string(msg))
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("El cliente normal no recibió el mensaje de trigger.")
	}

	select {
	case unexpectedMsg := <-normalClient.send:
		t.Errorf("Se recibió un mensaje inesperado: %s", string(unexpectedMsg))
	default:
		// Correcto, el canal debe estar vacío.
	}
}

func TestChatRoomConcurrency_RaceCondition(t *testing.T) {
	room := newChatRoom()
	go room.run()
	defer func() { room.quit <- true }()
	var wg sync.WaitGroup
	numClients := 100
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			client := newMockClient("user" + strconv.Itoa(i))
			room.register <- client
			room.broadcast <- &broadcastMessage{data: []byte("a"), sender: client}
			room.broadcast <- &broadcastMessage{data: []byte("b"), sender: client}
		}(i)
	}
	wg.Wait()
}
