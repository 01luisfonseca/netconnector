#!/bin/bash
# scripts/renew-ca.sh
# Verifica la validez del CA y lo renueva si expira en menos de 90 días (7776000 seg)

set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"
CERTS_DIR="$DIR/certs"
mkdir -p "$CERTS_DIR"
cd "$CERTS_DIR"

CA_CERT="ca.crt"
CA_KEY="ca.key"
DAYS_VALID=3650
SECONDS_BEFORE_RENEWAL=7776000 # 90 días

echo "=========================================="
echo "🛡️ Verificando Autoridad Certificadora (CA)..."
echo "=========================================="

needs_renewal=false

if [ ! -f "$CA_CERT" ] || [ ! -f "$CA_KEY" ]; then
    echo "⚠️ CA no encontrada. Se generará una nueva."
    needs_renewal=true
else
    # Verificar caducidad
    if ! openssl x509 -checkend $SECONDS_BEFORE_RENEWAL -noout -in "$CA_CERT" >/dev/null 2>&1; then
        echo "⏳ El CA actual expirará pronto. Iniciando renovación..."
        needs_renewal=true
    else
        echo "✅ El CA actual es válido por más de 90 días."
    fi
fi

if [ "$needs_renewal" = true ]; then
    echo "🔄 Generando CA..."
    openssl genrsa -out $CA_KEY 4096
    openssl req -new -x509 -days $DAYS_VALID -key $CA_KEY -out $CA_CERT -subj "/C=US/ST=State/L=City/O=Netconnector/CN=Netconnector CA"
    echo "✅ CA generada y guardada (Válida por $DAYS_VALID días)."
fi
