# Netconnector (Reverse Tunnel)

**Netconnector** es un sistema de túnel inverso escrito 100% en Go que permite exponer aplicaciones locales (localhost) a través de un VPS público con soporte para múltiples clientes y subdominios.

## Características

- **Multiplexación de Canales**: Utiliza un solo stream gRPC bidireccional por cliente.
- **Enrutamiento por Subdominio**: Mapea dinámicamente un Host HTTP (vía Nginx/Proxy) al Agente Local correspondiente.
- **Persistencia en SQLite**: Almacena las relaciones `subdominio -> client_id` en una base de datos local para seguridad.
- **Herramienta CLI Admin**: Permite gestionar mapeos (añadir/quitar) sin reiniciar el servidor.
- **Resiliencia**: gRPC KeepAlive integrado para detectar caídas de red y reconexión automática.

## Estructura del Proyecto

```text
netconnector/
├── bin/              # Binarios compilados
├── cmd/
│   ├── server/       # Servidor VPS (Bridge)
│   ├── client/       # Agente Local
│   ├── admin/        # CLI Administrativa
│   └── dummy/        # Servidor de pruebas (Echo)
├── internal/         # Lógica privada (Reverse Proxy, gRPC, DB)
├── proto/            # Definiciones de Protocol Buffers
├── specs/            # Especificaciones técnicas (Documentación)
└── .agents/          # Contexto para asistentes de IA
```

## Requisitos

- **Go** (1.22+)
- **Protoc** (para regenerar el código de gRPC)

## Instalación y Construcción

Para compilar todos los componentes del sistema:

```bash
# Instalar dependencias de gRPC (si es necesario)
make setup-deps

# Generar el código de Protobuf
make proto

# Construir todos los binarios localmente
make build
```

## Generación de Paquetes Portables (Releases)

Para generar archivos listos para enviar a tu VPS o Raspberry Pi:

```bash
# Generar ZIP para el Cliente (Agente + .env.example)
make release-client

# Generar ZIP para el Servidor (Bridge + Admin CLI + .env.example + scripts de mantenimiento)
make release-server
```
*El script detectará automáticamente tu sistema y te pedirá seleccionar la arquitectura de destino.*

## 🔒 Gestión de Certificados (Seguridad)

Netconnector incluye un sistema automatizado para gestionar TLS/SSL mediante una CA propia:

```bash
# 1. Generar certificados iniciales
./scripts/generate-certs.sh

# 2. Configurar renovación automática en Linux (Cron)
./scripts/setup-cron.sh
```
*Esto configurará una tarea semanal para renovar el certificado del servidor antes de que expire.*

## Guía Rápida de Uso

### 1. Iniciar el Servidor VPS

El servidor soporta configuración vía `.env`. 
- **Desde código fuente:** Copia `.env.server.example`.
- **Desde release (.zip):** Copia `.env.example`.

```bash
cp .env.example .env  # O .env.server.example si estás en desarrollo
# Edita .env con tus puertos y rutas de DB/Certs
./server  # O ./bin/server si estás en desarrollo
```
*(Opcionalmente sigue usando variables de entorno: `HTTP_PORT=8080 ./server`)*

### 2. Registrar un Cliente (Vía Admin CLI)
Asocia un subdominio (Host) a un ID de cliente específico:
```bash
./bin/admin add app1.local CLIENT-123
```

### 3. Iniciar el Agente Local

El agente puede configurarse mediante un archivo `.env` en la raíz del proyecto. Copia el ejemplo para empezar:

```bash
cp .env.example .env
# Edita .env con tus valores reales
./bin/client
```

### 4. Probar el Túnel
Si tienes tu aplicación local corriendo, ahora puedes acceder a ella a través del VPS:
```bash
curl -H "Host: app1.local" http://vps-ip:8080/tu-endpoint
```

## Desarrollo y Pruebas Rápidas (E2E)

Para probar localmente el flujo completo, he integrado comandos directos en el `Makefile`. 

⚠️ **IMPORTANTE:** El servidor, el dummy y el cliente deben ejecutarse **simultáneamente**. No presiones `Ctrl+C` para detenerlos, mejor abre **4 terminales (o pestañas fijas)** distintas:

**Terminal 1:** Iniciar el Dummy Echo Server (App destino en puerto 3000)
```bash
make run-dummy
```

**Terminal 2:** Iniciar el Bridge Server (VPS en puerto 8080 y gRPC en 50051)
```bash
make run-server
```

**Terminal 3:** Registrar un subdominio de prueba y correr el Agente Local
```bash
make setup-test       # Registra app1.local -> CLIENT-123
# Asegúrate de tener CLIENT_ID=CLIENT-123 en tu .env para el test
make run-client-local # Levanta el cliente leyendo el .env
```

**Terminal 4:** Disparar la prueba final (Consumidor HTTP)
```bash
curl -H "Host: app1.local" http://localhost:8080/endpoint-de-prueba
```

*Verás cómo el log de cada terminal se reporta en tiempo real cruzando la petición a través del túnel.*

## Licencia
Este proyecto es privado pero sigue los estándares de diseño de Go Standard Layout.
