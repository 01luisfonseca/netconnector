#!/bin/bash
# scripts/generate-certs.sh
# Genera certificados auto-firmados (CA propia) para Netconnector

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"
CERTS_DIR="$DIR/certs"

mkdir -p "$CERTS_DIR"
cd "$CERTS_DIR"

echo "=========================================="
echo "🛡️ Generando Autoridad Certificadora (CA) propia..."
echo "=========================================="

# 1. Generar la llave privada del CA
openssl genrsa -out ca.key 4096

# 2. Generar el certificado público del CA (Válido por 10 años)
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt -subj "/C=US/ST=State/L=City/O=Netconnector/CN=Netconnector CA"
echo "✅ CA generada: ca.key (Privada) y ca.crt (Pública)"
echo ""

echo "=========================================="
echo "🔐 Generando Certificado para Servidor VPS..."
echo "=========================================="

# 3. Generar la llave privada del Servidor
openssl genrsa -out server.key 4096

# 4. Generar la Solicitud de Firma de Certificado (CSR)
# Puedes cambiar el CN=localhost por tu dominio real si lo prefieres
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=State/L=City/O=Netconnector/CN=localhost"

# 5. Configurar extensiones para que sirva como certificado de servidor y tenga IP SAN
cat > extfile.cnf << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
EOF

# 6. Firmar el certificado del Servidor usando nuestro CA (Válido por 1 año)
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -extfile extfile.cnf

echo "🧹 Limpiando archivos temporales..."
rm server.csr extfile.cnf ca.srl

echo ""
echo "✅ Certificados generados exitosamente en la carpeta 'certs/' :"
echo ""
echo "▶️ Para el Servidor (VPS):"
echo "  - server.crt"
echo "  - server.key"
echo ""
echo "▶️ Para los Clientes (Agente Local):"
echo "  - ca.crt (Distribuir este archivo a los clientes para que confíen en el servidor)"
echo "⚠️  IMPORTANTE: Nunca distribuyas ni subas al repositorio los archivos .key!"
echo "=========================================="
