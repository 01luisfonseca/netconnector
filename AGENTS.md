# Netconnector - AI Context & Documentation

Este archivo proporciona el contexto necesario para que un asistente de IA entienda la arquitectura, el stack tecnológico y las reglas de negocio del proyecto **Netconnector**.

## 📍 Mapa del Proyecto (Rutas de Interés)

Para realizar tareas de mantenimiento o expansión, consulta los siguientes directorios y archivos:

- **Contrato gRPC**: [`proto/tunnel.proto`](file:///Users/andy/Desarrollo/netconnector/proto/tunnel.proto)
- **Lógica del Servidor VPS**: [`internal/server/`](file:///Users/andy/Desarrollo/netconnector/internal/server/)
  - *Proxy HTTP*: [`internal/server/proxy.go`](file:///Users/andy/Desarrollo/netconnector/internal/server/proxy.go)
  - *Base de Datos (SQLite)*: [`internal/server/db.go`](file:///Users/andy/Desarrollo/netconnector/internal/server/db.go)
  - *Manejo de gRPC*: [`internal/server/tunnel.go`](file:///Users/andy/Desarrollo/netconnector/internal/server/tunnel.go)
- **Lógica del Agente Local**: [`internal/client/`](file:///Users/andy/Desarrollo/netconnector/internal/client/)
  - *Ejecutor local*: [`internal/client/executor.go`](file:///Users/andy/Desarrollo/netconnector/internal/client/executor.go)
  - *Conectividad gRPC*: [`internal/client/agent.go`](file:///Users/andy/Desarrollo/netconnector/internal/client/agent.go)
- **Traducción HTTP-gRPC**: [`internal/shared/httpconv/httpconv.go`](file:///Users/andy/Desarrollo/netconnector/internal/shared/httpconv/httpconv.go)
- **Especificaciones Técnicas**: [`specs/TECHNICAL_SPEC.md`](file:///Users/andy/Desarrollo/netconnector/specs/TECHNICAL_SPEC.md)

## 📖 Descripción del Proyecto
**Netconnector** es un sistema de "Reverse Tunneling" escrito en Golang que permite exponer servicios locales (detrás de NAT/Firewall) al internet público a través de un VPS puente. Utiliza gRPC con streams bidireccionales para mantener conexiones persistentes y multiplexar peticiones HTTP.

## 🛠️ Stack Tecnológico
- **Lenguaje**: Go (Golang)
- **Comunicación**: gRPC (Google Remote Procedure Call)
- **Serialización**: Protocol Buffers (proto3)
- **Base de Datos**: SQLite (para mapeo de subdominios a Client IDs)
- **Logging**: `pkg/logger` (basado en `log/slog`)
- **Gestión de IDs**: UUID v7 (para request_id)

## 🌐 Reglas de Idioma (Estrictas)
1.  **Código Fuente**: Todo el código (nombres de variables, funciones, paquetes, tipos), comentarios técnicos dentro del archivo y logs deben estar exclusivamente en **Inglés**.
2.  **Comunicación con el Usuario**: Planes de implementación, walkthroughs, documentación de alto nivel y este archivo de contexto deben estar en **Español**.

## 🏗️ Arquitectura de Carpetas (Go Standard Layout)
- `cmd/`: Puntos de entrada para los binarios (Server, Client, Admin CLI).
- `internal/`: Lógica privada del negocio (Servidor, Cliente, Compartido).
- `pkg/`: Librerías públicas reutilizables (Logging).
- `proto/`: Definiciones de Protobuf y código generado (`pb/`).
- `specs/`: Documentación técnica detallada.

## 🔁 Flujo de Trabajo
1. El **Servidor VPS** escucha en `HTTP (8080)` y `gRPC (50051)`.
2. El **Cliente Local** se conecta vía gRPC enviando un `client_id` único e inmutable.
3. El **Administrador** mapea un subdominio a un `client_id` en SQLite usando la CLI `admin`.
4. Una petición HTTP externa llega al VPS -> Se busca el `client_id` asociado -> Se envía por el stream gRPC con un `request_id` -> El cliente local ejecuta el HTTP localmente -> Devuelve la respuesta por el mismo stream gRPC.

## 💡 Principios de Desarrollo
- **Multiplexación**: Todas las peticiones de un cliente viajan por el mismo stream gRPC usando goroutines para ejecución paralela.
- **Resiliencia**: Uso de gRPC KeepAlive para detectar caídas de red y reconexión automática en el cliente (Agente).
- **Seguridad**: El mapeo es controlado por el administrador (Whitelist).
