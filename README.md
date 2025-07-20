# Reto #4: Aplicación de Chat en Tiempo Real con WebSockets en Go

Este proyecto implementa el backend de un sistema de chat en tiempo real utilizando Go, WebSockets y primitivas de concurrencia nativas. La aplicación soporta múltiples clientes concurrentes en una única sala de chat global.

## Cómo Ejecutar

1.  **Requisitos:**
    *   Tener Go instalado (versión 1.18 o superior).

2.  **Instalación:**
    Clona el repositorio y navega al directorio del proyecto. Luego, instala las dependencias:
    ```bash
    git clone <url-del-repositorio>
    cd chat-app
    go mod tidy
    ```

3.  **Ejecutar el Servidor:**
    ```bash
    go run .
    ```
    El servidor se iniciará en `http://localhost:8080`.

4.  **Probar el Chat:**
    *   Abre `http://localhost:8080` en varias pestañas o navegadores.
    *   En cada pestaña, introduce un nombre de usuario único y únete al chat.
    *   Envía mensajes y observa cómo se difunden en tiempo real a todos los clientes conectados.
    *   Cierra una pestaña para ver el mensaje de desconexión.

5.  **Ejecutar Pruebas:**
    Para verificar la correctitud y la seguridad de la concurrencia, ejecuta las pruebas unitarias con el detector de carreras:
    ```bash
    go test -v -race ./...
    ```
    Un resultado exitoso sin errores de "race" indica que la implementación es robusta.

## Arquitectura Concurrente

La arquitectura se basa en el patrón "Hub and Spoke" (o Hub y Clientes), que es ideal para aplicaciones de difusión.

1.  **El Hub (`hub.go`):**
    *   Es el componente central que actúa como la sala de chat.
    *   Se ejecuta en su **propia goroutine** (`go hub.run()`) y funciona como un bucle de eventos.
    *   Utiliza canales (`register`, `unregister`, `broadcast`) para recibir "comandos" de los clientes de manera secuencial y segura. Esto centraliza todas las modificaciones al estado compartido (el mapa de clientes), eliminando la necesidad de bloqueos en la lógica de negocio principal del hub.

2.  **Los Clientes (`client.go`):**
    *   Cada cliente conectado es representado por una `struct Client`.
    *   Por cada cliente, se lanzan **dos goroutines**:
        *   `readPump()`: Lee continuamente los mensajes entrantes de la conexión WebSocket del cliente. Cuando recibe un mensaje, lo envía al canal `broadcast` del Hub.
        *   `writePump()`: Escucha en un canal de envío (`send`) específico del cliente. Cuando el Hub difunde un mensaje, lo coloca en este canal, y el `writePump` se encarga de escribirlo en la conexión WebSocket.
    *   Este diseño de dos goroutines por cliente desacopla la lectura de la escritura, evitando que un cliente lento para leer bloquee la escritura de mensajes hacia él, y viceversa.

### Seguridad Concurrente (`sync.RWMutex`)

El estado compartido principal es el mapa de clientes (`map[*Client]bool`) en el Hub. Para protegerlo de accesos concurrentes peligrosos, se utiliza un `sync.RWMutex`:

*   **Escrituras (`Lock()`):** Cuando un cliente se registra (`registerClient`) o se da de baja (`unregisterClient`), se adquiere un bloqueo de escritura (`h.clientsRWMutex.Lock()`). Esto garantiza que solo una goroutine pueda modificar el mapa a la vez, previniendo condiciones de carrera.
*   **Lecturas (`RLock()`):** Cuando el Hub difunde un mensaje (`broadcastMessage`), itera sobre el mapa de clientes. Para esta operación, se adquiere un bloqueo de lectura (`h.clientsRWMutex.RLock()`). Un `RWMutex` permite que **múltiples lectores** accedan al mapa simultáneamente siempre que no haya ningún escritor, lo cual es perfecto para la difusión, ya que es una operación de solo lectura y muy frecuente.

### Manejo de Conexiones y Desconexiones

*   **Conexión:** El manejador HTTP `/ws` actualiza la conexión, crea una nueva instancia de `Client` y la envía al canal `register` del Hub. El Hub la añade a su mapa de clientes y difunde un mensaje de "unión".
*   **Desconexión:** La desconexión se detecta en `readPump()`. Cuando `conn.ReadMessage()` devuelve un error (por ejemplo, el cliente cierra la pestaña), la goroutine `readPump` se termina. Un `defer` se encarga de la limpieza: envía el cliente al canal `unregister` del Hub y cierra la conexión WebSocket. El Hub, al recibir la señal de `unregister`, elimina al cliente de su mapa, cierra su canal `send` (lo que a su vez termina la `writePump` del cliente) y difunde un mensaje de "abandono".

### Elección del Paquete WebSocket: `github.com/gorilla/websocket`

Se eligió `gorilla/websocket` sobre la librería estándar `golang.org/x/net/websocket` por su API más robusta, mejor manejo de errores (ej. `IsUnexpectedCloseError`), control explícito sobre los tipos de mensaje (ping/pong) y su popularidad y mantenimiento activo en la comunidad, lo que la convierte en una opción más segura y completa para aplicaciones de producción.