package main

import (
	"sync"
	"testing"
	"time"
)

func newMockClient(username string) *Client {
	return &Client{
		send:     make(chan []byte, 256),
		username: username,
	}
}

func TestHubRegisterAndUnregister(t *testing.T) {
	hub := newHub()
	go hub.run()
	defer func() { hub.quit <- true }()

	client := newMockClient("testuser")

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.clientsRWMutex.RLock()
	if len(hub.clients) != 1 {
		t.Errorf("Se esperaba 1 cliente, se obtuvieron %d", len(hub.clients))
	}
	if _, ok := hub.clients[client]; !ok {
		t.Error("El cliente no fue registrado correctamente")
	}
	hub.clientsRWMutex.RUnlock()

	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	hub.clientsRWMutex.RLock()
	if len(hub.clients) != 0 {
		t.Errorf("Se esperaba 0 clientes, se obtuvieron %d", len(hub.clients))
	}
	hub.clientsRWMutex.RUnlock()
}

func TestHubBroadcast(t *testing.T) {
	hub := newHub()
	go hub.run()
	defer func() { hub.quit <- true }()
	client1 := newMockClient("user1")
	client2 := newMockClient("user2")

	hub.register <- client1
	hub.register <- client2
	time.Sleep(50 * time.Millisecond)

	<-client1.send
	<-client1.send
	<-client2.send
	<-client2.send

	message := []byte("hola a todos")
	hub.broadcast <- message

	select {
	case received1 := <-client1.send:
		if string(received1) != string(message) {
			t.Errorf("Client1 recibió '%s', se esperaba '%s'", string(received1), string(message))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client1 no recibió el mensaje a tiempo")
	}

	select {
	case received2 := <-client2.send:
		if string(received2) != string(message) {
			t.Errorf("Client2 recibió '%s', se esperaba '%s'", string(received2), string(message))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client2 no recibió el mensaje a tiempo")
	}
}

func TestHubConcurrency_RaceCondition(t *testing.T) {
	hub := newHub()
	go hub.run()
	defer func() { hub.quit <- true }()

	var wg sync.WaitGroup
	numClients := 100
	numMessages := 10

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := newMockClient("user")
			hub.register <- client

			for j := 0; j < numMessages; j++ {
				hub.broadcast <- []byte("mensaje")
				time.Sleep(time.Millisecond * time.Duration(1+j%5))
			}

			time.Sleep(50 * time.Millisecond)
			hub.unregister <- client
		}()
	}

	wg.Wait()
	t.Log("Prueba de concurrencia completada sin condiciones de carrera detectadas.")
}
