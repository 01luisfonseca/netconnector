# Technical Specification - Netconnector

Este documento detalla la lógica interna, el modelo de datos y la arquitectura técnica del sistema de Túnel Inverso.

## 1. Arquitectura de gRPC (Multiplexación)

Utilizamos un servicio gRPC con streams bidireccionales para permitir la comunicación "Push" desde el servidor público hacia el cliente local.

### TunnelService
- **`TunnelStream`**: Un flujo asíncrono que transporta mensajes de tipo `TunnelMessage`.
- **`TunnelMessage`**: Un wrapper `oneof` que encapsula:
  - `RegisterRequest` / `RegisterResponse`: Handshake para establecer la identidad del cliente.
  - `HTTPRequest` / `HTTPResponse`: Encapsulación de tráfico HTTP nativo.

### Identificación y UUID
Cada petición HTTP recibida en el VPS genera un `request_id` (UUID v7). Este ID se usa para:
- Mapear la respuesta asíncrona que llega por el stream al "handler" HTTP que originó la petición.
- Garantizar que el cliente local sepa a qué hilo o goroutine de ejecución pertenece la respuesta.

## 2. Persistencia (SQLite)

El servidor VPS utiliza SQLite para gestionar el acceso dinámico de los clientes.

### Estructura de la Tabla `mappings`
```sql
CREATE TABLE mappings (
    subdomain TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```
- **`subdomain`**: El Host HTTP que llega al proxy (ej: `app1.dominio.com`).
- **`client_id`**: El identificador inmutable que el cliente local envía en su registro.

## 3. Estrategia de Resiliencia

### gRPC KeepAlive
Se configuran parámetros de KeepAlive en ambos extremos (Server y Client):
- **Server**: Envía pings cada 30 segundos. Si no hay respuesta en 10s, cierra la conexión.
- **Client**: Intentará reconexión automática con un reintento de 5 segundos tras una caída del stream.

### Timeouts de Aplicación
1. **Frente Público**: 30 segundos de espera por una respuesta del túnel antes de retornar `504 Gateway Timeout`.
2. **Frente Local**: El agente local tiene un timeout de 15 segundos para las peticiones contra el backend local (`localhost:3000`).

## 4. Traducción HTTP <-> gRPC (`httpconv`)

La librería interna `httpconv` maneja la conversión de objetos `http.Request` y `http.Response` de Go a sus equivalentes en Protobuf. Esto asegura que headers como `Content-Type`, `Authorization` y `User-Agent` se propaguen íntegramente a través del túnel.

## 5. Límites de Tamaño (Payload)

Se ha establecido un límite teórico de **10MB** para peticiones y respuestas mediante la configuración de `MaxCallRecvMsgSize` en gRPC. Esto es suficiente para el envío de documentos PDF, imágenes pesadas y payloads JSON extensos sin la complejidad de un streaming de bytes fraccionado.

## 6. Seguridad y Autorización (Zero-Trust)

La arquitectura asegura que la comunicación a través de internet sea privada y estrictamente controlada.

### Autorización del Cliente (Handshake)
Al iniciar la conexión `TunnelStream`, el cliente local debe enviar un mensaje `RegisterRequest` con su `client_id`. El Servidor VPS intercepta este paso y consulta la tabla SQLite `mappings` (`IsClientIDRegistered`). Si el ID no ha sido emparejado previamente con un subdominio por el administrador, el VPS retorna un código de error gRPC `PermissionDenied` y cierra el stream inmediatamente, bloqueando accesos no registrados.

### Encriptación gRPC (TLS con Autoridad Propia)
El sistema soporta el despliegue de **certificados auto-firmados (CA propio)** para blindar el canal bidireccional contra ataques Man-in-the-Middle.
- **Servidor:** Lee las variables `TLS_CERT_FILE` y `TLS_KEY_FILE` al inicializar `grpc.NewServer()`. Al estar presentes, la comunicación en texto plano se desactiva.
- **Cliente:** Lee la variable `TLS_CA_FILE` para inyectar explícitamente la Autoridad Certificadora en el esquema de confianza de gRPC mediante `credentials.NewClientTLSFromFile`, aceptando así el certificado auto-firmado del Servidor VPS de forma segura. Existe un flag seguro de evasión (`GRPC_INSECURE=true`) exclusivamente para debugging/desarrollo local.
- **Herramientas:** El script `scripts/generate-certs.sh` gestiona la creación de estas llaves criptográficas (CA, Certificado Servidor y Llave Privada) en una operación manual.
