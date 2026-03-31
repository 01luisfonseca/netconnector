# Guía de Configuración: Seguridad TLS y Control de Acceso (gRPC)

Netconnector implementa seguridad en dos frentes para proteger la comunicación entre tus Agentes Locales y el Servidor VPS:
1. **Autorización Fuerte**: Solo los `client_id` pre-registrados en la base de datos (mediante la CLI de `admin`) pueden conectarse al servidor VPS.
2. **Encriptación de Red (TLS)**: La comunicación gRPC entre el cliente y el servidor puede (y debe) ser encriptada usando certificados.

Este documento enumera los pasos para activar la **Encriptación con Certificados Auto-Firmados**.

---

## 1. Generar los Certificados Propios

Hemos preparado un script que automatiza la creación de una Autoridad Certificadora (CA) propia y la emisión de certificados firmados por esta.

1. Sitúate en la raíz del proyecto.
2. Ejecuta el script:
   ```bash
   ./scripts/generate-certs.sh
   ```
3. Se creará una carpeta `/certs` con los siguientes archivos clave:
   - `ca.crt`: **[PÚBLICO]** Certificado Maestro que el Agente necesita para "confiar" en tu VPS.
   - `server.crt`: **[PÚBLICO]** Certificado del Servidor, firmado por el CA.
   - `server.key`: **[SECRETO]** Llave privada del Servidor. ¡No la compartas!

---

## 2. Iniciar el Servidor VPS (Seguro)

Para iniciar el servidor activado con TLS, simplemente pásale por variables de entorno las rutas a su certificado y su llave:

```bash
# Exportar las rutas a los certificados generados
export TLS_CERT_FILE=./certs/server.crt
export TLS_KEY_FILE=./certs/server.key

# Iniciar el servidor
./bin/server
```
*(Si no provees estas variables, el servidor arrancará en modo Inseguro - Texto Plano, solo útil para desarrollo).*

---

## 3. Iniciar el Agente Local (Seguro)

> ⚠️ IMPORTANTE: Si levantas el Agente Local en una **máquina distinta** a la del VPS, debes copiar manualmente el archivo `certs/ca.crt` (el tuyo, generado en el Paso 1) a la máquina del Agente.

Para que el Agente pueda establecer una comunicación confiable con tu VPS, indícale dónde está el archivo de la CA:

```bash
# Indicar el archivo CA de confianza
export TLS_CA_FILE=/ruta/donde/tengas/ca.crt

# Opcional (Depende de cómo generaste el certificado)
# export SERVER_ADDR=vps.midominio.com:50051  <- Ideal
export SERVER_ADDR=localhost:50051

# Iniciar el agente
./bin/agent
```

### ¿Y si el servidor está en Inseguro o quiero debuggear?
El cliente por defecto **EXIGE** seguridad TLS e intentará validarla (ya sea contra tu `ca.crt` si se lo pasas, o contra la libreta de Let's Encrypt de tu Sistema Operativo). 

Si estás desarrollando en local y no quieres lidiar con certificados en absoluto, debes desactivarlo explícitamente:
```bash
export GRPC_INSECURE=true
./bin/agent
```
