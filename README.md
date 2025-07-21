# Reto #4: Aplicación de Chat en Tiempo Real con WebSockets

Este proyecto implementa el backend para una aplicación de chat simple y robusta utilizando Go, WebSockets y primitivas de concurrencia, adhiriéndose estrictamente a las especificaciones del reto. Permite que múltiples usuarios se conecten a una sala de chat global y se comuniquen en tiempo real.

## Arquitectura Concurrente

La arquitectura implementada sigue un modelo híbrido que combina canales y mutexes para garantizar la concurrencia segura, tal como se especificó en los requerimientos del reto.

1.  **El Hub (`chat_room.go`):**
    *   Actúa como la sala de chat central, ejecutándose en su propia goroutine (`go room.run()`).
    *   **Canales para Comunicación:** Utiliza canales (`register`, `unregister`, `broadcast`) para recibir *eventos* o *comandos* de las goroutines de los clientes. Esto desacopla a los clientes del hub y organiza el flujo de trabajo.
    *   **`sync.RWMutex` para Protección de Estado:** El estado compartido (el mapa `clients`) está explícitamente protegido por un `sync.RWMutex`.
        *   Las operaciones de **escritura** (registrar o desregistrar un cliente) utilizan un bloqueo completo (`Lock`/`Unlock`) para garantizar un acceso exclusivo.
        *   La operación de **lectura** (iterar sobre los clientes para difundir un mensaje) utiliza un bloqueo de lectura (`RLock`/`RUnlock`), que es más permisivo y permite múltiples lecturas concurrentes si la arquitectura lo necesitara, cumpliendo así con el espíritu del requerimiento.

2.  **Los Clientes (`client.go`):**
    *   Cada cliente conectado es representado por una `struct Client` y gestionado por dos goroutines: `readPump` (para leer mensajes del cliente) y `writePump` (para enviar mensajes al cliente).

Esta combinación de primitivas demuestra la capacidad de utilizar diferentes herramientas de concurrencia de Go para distintos propósitos: canales para el flujo de mensajes y eventos, y mutexes para la protección directa de datos críticos compartidos.

## Seguridad y Robustez

*   **Fuente Única de Verdad (Single Source of Truth):** El frontend (`index.html`) está diseñado para tratar al servidor como la única fuente de verdad, garantizando una vista consistente del chat para todos los participantes.
*   **Sanitización de Entradas:** Se sanitizan las entradas del usuario tanto en el backend (`html.EscapeString`) como en el frontend para prevenir ataques XSS.
*   **Manejo de Desconexiones:** La aplicación maneja elegantemente las desconexiones mediante la detección de errores en `readPump` y el uso de un sistema de ping/pong para limpiar conexiones inactivas.
*   **Pruebas Robustas:** La suite de pruebas (`chat_room_test.go`) valida la lógica de la aplicación y utiliza `go test -race` para confirmar que la implementación de la concurrencia es segura y no produce condiciones de carrera.

## Elección del Paquete WebSocket: `gorilla/websocket`

Se optó por utilizar la librería `github.com/gorilla/websocket` por su API robusta, control detallado sobre la conexión (timeouts, pings/pongs) y su estatus como estándar de facto en la comunidad de Go, lo que asegura un buen mantenimiento y amplia documentación.