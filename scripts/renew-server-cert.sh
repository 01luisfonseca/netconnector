#!/bin/bash
# scripts/renew-server-cert.sh
# Verifica la validez del certificado del servidor y lo renueva si expira en menos de 30 días (2592000 seg)

set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"
CERTS_DIR="$DIR/certs"
mkdir -p "$CERTS_DIR"
cd "$CERTS_DIR"

IP_FILE=".server_ip"

# 1. Si se pasa por variable de entorno, guardarla
if [ -n "$SERVER_IP" ]; then
    echo "$SERVER_IP" > "$IP_FILE"
    echo "📍 IP del servidor establecida desde variable de entorno: $SERVER_IP"
# 2. Si no, intentar leerla del archivo persistente
elif [ -f "$IP_FILE" ]; then
    SERVER_IP=$(cat "$IP_FILE")
    echo "📖 IP del servidor cargada desde persistencia: $SERVER_IP"
else
    echo "⚠️ No se definió SERVER_IP y no existe archivo de persistencia ($IP_FILE)."
    echo "💡 Se generará el certificado sin IP externa (solo localhost)."
fi

SERVER_CERT="server.crt"
SERVER_KEY="server.key"
CA_CERT="ca.crt"
CA_KEY="ca.key"

DAYS_VALID=365
SECONDS_BEFORE_RENEWAL=2592000 # 30 días

echo "=========================================="
echo "🔐 Verificando Certificado de Servidor VPS..."
echo "=========================================="

# Primero nos aseguramos de que el CA exista y sea válido
"$DIR/scripts/renew-ca.sh"

needs_renewal=false

if [ ! -f "$SERVER_CERT" ] || [ ! -f "$SERVER_KEY" ]; then
    echo "⚠️ Certificado de servidor no encontrado. Se generará uno nuevo."
    needs_renewal=true
else
    if ! openssl x509 -checkend $SECONDS_BEFORE_RENEWAL -noout -in "$SERVER_CERT" >/dev/null 2>&1; then
        echo "⏳ El certificado expirará en menos de 30 días. Iniciando renovación..."
        needs_renewal=true
    else
        # Verificar que haga match con la CA actual
        verify=$(openssl verify -CAfile "$CA_CERT" "$SERVER_CERT" 2>/dev/null | grep "OK" || true)
        if [ -z "$verify" ]; then
            echo "⚠️ El certificado del servidor no embona o no fue firmado por el CA actual. Se regenerará."
            needs_renewal=true
        else
            echo "✅ El certificado actual del servidor es válido."
        fi
    fi
fi

if [ "$needs_renewal" = true ]; then
    echo "🔄 Generando nuevo certificado de Servidor..."
    openssl genrsa -out $SERVER_KEY 4096
    openssl req -new -key $SERVER_KEY -out server.csr -subj "/C=US/ST=State/L=City/O=Netconnector/CN=localhost"
    
    cat > extfile.cnf << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
$( [ -n "$SERVER_IP" ] && echo "IP.2 = $SERVER_IP" )
EOF

    openssl x509 -req -in server.csr -CA $CA_CERT -CAkey $CA_KEY -CAcreateserial -out $SERVER_CERT -days $DAYS_VALID -extfile extfile.cnf
    rm server.csr extfile.cnf
    [ -f ca.srl ] && rm ca.srl
    echo "✅ Certificado de Servidor regenerado exitosamente."

    # Intentar reiniciar la aplicación usando el comando configurado
    RESTART_FILE="$DIR/scripts/.restart_cmd"
    if [ -f "$RESTART_FILE" ]; then
        RESTART_CMD=$(cat "$RESTART_FILE")
        if [ -n "$RESTART_CMD" ]; then
            echo "🚀 Ejecutando comando de reinicio (Service reload)..."
            echo "Comando: $RESTART_CMD"
            eval "$RESTART_CMD" || echo "⚠️ Error al ejecutar el comando de reinicio."
        fi
    fi
fi
