# Reto #4: Aplicación de Chat en Tiempo Real con WebSockets

Este proyecto implementa el backend para una aplicación de chat simple y robusta utilizando Go, WebSockets y primitivas de concurrencia. Permite que múltiples usuarios se conecten a una sala de chat global y se comuniquen en tiempo real.

## Arquitectura Concurrente

La arquitectura se basa en el patrón "Hub and Spoke" (o Hub y Clientes), siguiendo las mejores prácticas de concurrencia en Go.

1.  **El Hub (`chat_room.go`):**
    *   Actúa como la sala de chat central. Se ejecuta en su **propia goroutine** (`go room.run()`) y procesa todos los eventos de manera secuencial a través de un bucle `select`.
    *   Utiliza **canales** (`register`, `unregister`, `broadcast`) para recibir comandos de las goroutines de los clientes. Este es el núcleo del diseño de concurrencia: al serializar todas las operaciones que modifican el estado compartido (el mapa `clients`) a través de una única goroutine, **se eliminan por diseño las condiciones de carrera**. No es necesario un `sync.Mutex` porque ningún otro proceso puede acceder al mapa directamente.
    *   Este enfoque sigue el principio idiomático de Go: *"Do not communicate by sharing memory; instead, share memory by communicating."*

2.  **Los Clientes (`client.go`):**
    *   Cada cliente conectado es representado por una `struct Client`.
    *   Por cada cliente, se lanzan dos goroutines:
        *   `readPump()`: Lee continuamente los mensajes del WebSocket del cliente, los procesa (añadiendo metadatos como `username` y `timestamp`) y los envía al canal `broadcast` del Hub.
        *   `writePump()`: Escucha en un canal de envío (`send`) específico del cliente, donde el Hub coloca los mensajes que deben ser enviados a ese cliente.

## Seguridad y Robustez

*   **Fuente Única de Verdad (Single Source of Truth):** El frontend (`index.html`) está diseñado para tratar al servidor como la única fuente de verdad. Cuando un usuario envía un mensaje, este solo se muestra en su propia pantalla después de que el servidor lo ha procesado y lo ha redifundido. Esto garantiza que todos los participantes tengan una vista 100% consistente del historial del chat.
*   **Sanitización de Entradas:** Para prevenir vulnerabilidades de Cross-Site Scripting (XSS), todas las entradas del usuario (nombre de usuario) son sanitizadas en el backend con `html.EscapeString`. El frontend también escapa todo el contenido antes de insertarlo en el DOM.
*   **Manejo de Desconexiones:** La aplicación maneja elegantemente las desconexiones. `readPump` detecta cierres de conexión (normales o inesperados) y notifica al Hub para desregistrar al cliente. El uso de pings/pongs periódicos (`pingPeriod`, `pongWait`) asegura que las conexiones inactivas o rotas también se detecten y se limpien.
*   **Pruebas Robustas:** Las pruebas unitarias validan la lógica de registro, desregistro y difusión. Una prueba de concurrencia dedicada (`TestChatRoomConcurrency_RaceCondition`) simula la actividad de cientos de clientes simultáneamente, y se ejecuta con el flag `-race` para demostrar la ausencia de condiciones de carrera de datos.

## Elección del Paquete WebSocket: `gorilla/websocket`

Se optó por utilizar la librería `github.com/gorilla/websocket` en lugar del paquete estándar `golang.org/x/net/websocket`. La justificación es la siguiente:

1.  **API Más Robusta y Completa:** `gorilla/websocket` ofrece una API de más alto nivel, con funciones de ayuda que simplifican el manejo de conexiones, como `IsUnexpectedCloseError`, que facilita la distinción entre cierres de conexión normales y errores de red.
2.  **Control Detallado de la Conexión:** Proporciona un control más granular sobre los timeouts de lectura/escritura (`SetReadDeadline`, `SetWriteDeadline`), lo cual es fundamental para implementar pings de *keep-alive* y manejar conexiones inactivas.
3.  **Estándar de la Industria:** Es la librería de facto para WebSockets en la comunidad de Go, lo que garantiza un buen mantenimiento, una amplia base de usuarios y una gran cantidad de ejemplos y documentación.