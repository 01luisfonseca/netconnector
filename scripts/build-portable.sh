#!/bin/bash

# Script para construir paquetes portables de Netconnector (Cliente o Servidor).
# Basado en la arquitectura de empaquetado de Orato Go.

set -e # Detener en caso de error

# Colores para la terminal
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Validar argumento inicial
COMPONENT=$1
if [ "$COMPONENT" != "client" ] && [ "$COMPONENT" != "server" ]; then
    echo -e "${RED}Error: Debes especificar 'client' o 'server' como primer argumento.${NC}"
    echo "Uso: ./scripts/build-portable.sh [client|server]"
    exit 1
fi

echo -e "${BLUE}==================================================${NC}"
echo -e "${BLUE}   🚀 Netconnector - Generador Portable ($COMPONENT) ${NC}"
echo -e "${BLUE}==================================================${NC}"

# 1. Menú Interactivo de Sistema Operativo y Arquitectura
echo -e "\n${YELLOW}Seleccione el sistema operativo de destino:${NC}"
echo "1) Linux"
echo "2) Windows"
echo "3) macOS"
read -p "Opción (1-3): " OS_CHOICE

case $OS_CHOICE in
    1)
        TARGET_OS="linux"
        echo -e "\n${YELLOW}Seleccione la arquitectura para Linux:${NC}"
        echo "1) x64 (64-bit)"
        echo "2) ARM64 (Raspberry Pi 4/5, AWS Graviton)"
        echo "3) ARMv7 (Raspberry Pi 3/Viejas)"
        read -p "Arquitectura (1-3): " ARCH_CHOICE
        case $ARCH_CHOICE in
            1) TARGET_ARCH="amd64" ;;
            2) TARGET_ARCH="arm64" ;;
            3) TARGET_ARCH="arm"; export GOARM=7 ;;
            *) echo -e "${RED}Opción inválida.${NC}"; exit 1 ;;
        esac
        ;;
    2)
        TARGET_OS="windows"
        TARGET_ARCH="amd64" # Por defecto x64 para servidores
        echo -e "\n${YELLOW}Arquitectura seleccionada: Windows x64${NC}"
        ;;
    3)
        TARGET_OS="darwin"
        echo -e "\n${YELLOW}Seleccione la arquitectura para macOS:${NC}"
        echo "1) Apple Silicon (M1/M2/M3)"
        echo "2) Intel (64-bit)"
        read -p "Arquitectura (1-2): " ARCH_CHOICE
        case $ARCH_CHOICE in
            1) TARGET_ARCH="arm64" ;;
            2) TARGET_ARCH="amd64" ;;
            *) echo -e "${RED}Opción inválida.${NC}"; exit 1 ;;
        esac
        ;;
    *)
        echo -e "${RED}Opción inválida.${NC}"; exit 1 ;;
esac

# 2. Definir rutas y nombres
PROJECT_ROOT=$(pwd)
RELEASE_DIR="$PROJECT_ROOT/release"
TEMP_BUILD_DIR="$RELEASE_DIR/temp_$COMPONENT"
ZIP_NAME="netconnector-${COMPONENT}-${TARGET_OS}-${TARGET_ARCH}.zip"

# Nombre del binario base
BINARY_EXT=""
if [ "$TARGET_OS" == "windows" ]; then
    BINARY_EXT=".exe"
fi

echo -e "\n${GREEN}⚙️ Iniciando construcción para ${TARGET_OS}/${TARGET_ARCH}...${NC}"

# 3. Limpiar compilaciones previas
rm -rf "$TEMP_BUILD_DIR"
rm -f "$RELEASE_DIR/$ZIP_NAME"
mkdir -p "$TEMP_BUILD_DIR"

# 4. Compilación Cruzada (Go)
echo -e "${BLUE}🔨 Compilando binarios...${NC}"

if [ "$COMPONENT" == "client" ]; then
    GOOS=$TARGET_OS GOARCH=$TARGET_ARCH go build -o "$TEMP_BUILD_DIR/client$BINARY_EXT" cmd/client/main.go
    cp "$PROJECT_ROOT/.env.example" "$TEMP_BUILD_DIR/.env.example"
    echo -e "${GREEN}✅ Cliente compilado e incluido .env.example${NC}"
else
    GOOS=$TARGET_OS GOARCH=$TARGET_ARCH go build -o "$TEMP_BUILD_DIR/server$BINARY_EXT" cmd/server/main.go
    GOOS=$TARGET_OS GOARCH=$TARGET_ARCH go build -o "$TEMP_BUILD_DIR/admin$BINARY_EXT" cmd/admin/main.go
    
    # Archivos adicionales para el servidor
    cp "$PROJECT_ROOT/.env.server.example" "$TEMP_BUILD_DIR/.env.example"
    mkdir -p "$TEMP_BUILD_DIR/scripts"
    cp "$PROJECT_ROOT/scripts/generate-certs.sh" "$TEMP_BUILD_DIR/scripts/"
    cp "$PROJECT_ROOT/scripts/renew-ca.sh" "$TEMP_BUILD_DIR/scripts/"
    cp "$PROJECT_ROOT/scripts/renew-server-cert.sh" "$TEMP_BUILD_DIR/scripts/"
    cp "$PROJECT_ROOT/scripts/setup-cron.sh" "$TEMP_BUILD_DIR/scripts/"
    
    echo -e "${GREEN}✅ Servidor, Admin CLI y scripts de mantenimiento incluidos${NC}"
fi

# 5. Crear el ZIP
echo -e "${BLUE}🗜️ Creando paquete ZIP...${NC}"
cd "$TEMP_BUILD_DIR"
# Usamos zip si está disponible, o informamos
if command -v zip >/dev/null 2>&1; then
    zip -r "$RELEASE_DIR/$ZIP_NAME" .
else
    echo -e "${RED}Error: 'zip' no está instalado. No se pudo comprimir el paquete.${NC}"
    exit 1
fi

# 6. Limpieza final
cd "$PROJECT_ROOT"
rm -rf "$TEMP_BUILD_DIR"

echo -e "\n${GREEN}==================================================${NC}"
echo -e "${GREEN} ✅ ¡Paquete listo! ${NC}"
echo -e " Ubicación: ${RELEASE_DIR}/${ZIP_NAME}"
echo -e "${GREEN}==================================================${NC}"
