// hub_test.go
package main

import (
	"sync"
	"testing"
	"time"
)

// mockClient simula un cliente para pruebas sin una conexión de red real.
// No necesitamos redefinirlo, ya que el Client real funciona para nuestras pruebas de hub.

func newMockClient(username string) *Client {
	// Usamos un Client real pero sin una conexión `*websocket.Conn`.
	// Las pruebas se centran en la interacción con el Hub a través de canales.
	return &Client{
		send:     make(chan []byte, 256),
		username: username,
	}
}

// TestHubRegisterAndUnregister prueba el registro y la anulación de registro de clientes.
func TestHubRegisterAndUnregister(t *testing.T) {
	hub := newHub()
	go hub.run()
	defer func() { hub.quit <- true }() // Cierra el hub al final de la prueba

	client := newMockClient("testuser")

	// Registrar
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // Dar tiempo al hub para procesar

	hub.clientsRWMutex.RLock()
	if len(hub.clients) != 1 {
		t.Errorf("Se esperaba 1 cliente, se obtuvieron %d", len(hub.clients))
	}
	if _, ok := hub.clients[client]; !ok {
		t.Error("El cliente no fue registrado correctamente")
	}
	hub.clientsRWMutex.RUnlock()

	// Anular registro
	hub.unregister <- client
	time.Sleep(50 * time.Millisecond) // Dar tiempo al hub para procesar

	hub.clientsRWMutex.RLock()
	if len(hub.clients) != 0 {
		t.Errorf("Se esperaba 0 clientes, se obtuvieron %d", len(hub.clients))
	}
	hub.clientsRWMutex.RUnlock()
}

// TestHubBroadcast prueba la difusión de mensajes a múltiples clientes.
func TestHubBroadcast(t *testing.T) {
	hub := newHub()
	go hub.run()
	defer func() { hub.quit <- true }() // Cierra el hub al final de la prueba

	client1 := newMockClient("user1")
	client2 := newMockClient("user2")

	hub.register <- client1
	hub.register <- client2
	time.Sleep(50 * time.Millisecond) // Dar tiempo a que se procesen los registros

	// ******** INICIO DE LA SOLUCIÓN ********
	// LIMPIEZA: Los clientes reciben mensajes de unión.
	// Leemos y descartamos los mensajes que no nos interesan para esta prueba.
	// Cada cliente recibe 2 mensajes: el suyo y el del otro.
	<-client1.send
	<-client1.send
	<-client2.send
	<-client2.send
	// ******** FIN DE LA SOLUCIÓN ********

	// Ahora que los canales están limpios, enviamos el mensaje que SÍ queremos probar.
	message := []byte("hola a todos")
	hub.broadcast <- message

	// Verificar que ambos clientes recibieron el mensaje correcto
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

// TestHubConcurrency_RaceCondition simula alta concurrencia para detectar condiciones de carrera.
// Ejecuta esta prueba con el flag -race: `go test -race`
func TestHubConcurrency_RaceCondition(t *testing.T) {
	hub := newHub()
	go hub.run()
	// ******** LA LÍNEA QUE FALTABA ESTÁ AQUÍ ********
	defer func() { hub.quit <- true }() // Cierra el hub al final de la prueba

	var wg sync.WaitGroup
	numClients := 100
	numMessages := 10

	// Goroutines para registrar clientes concurrentemente
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := newMockClient("user")
			hub.register <- client

			// Cada cliente envía algunos mensajes
			for j := 0; j < numMessages; j++ {
				hub.broadcast <- []byte("mensaje")
				time.Sleep(time.Millisecond * time.Duration(1+j%5)) // Pequeña pausa aleatoria
			}

			// Desconectar al cliente después de un tiempo
			time.Sleep(50 * time.Millisecond)
			hub.unregister <- client
		}()
	}

	wg.Wait()
	// Si `go test -race` pasa sin errores, la prueba es un éxito.
	t.Log("Prueba de concurrencia completada sin condiciones de carrera detectadas.")
}
