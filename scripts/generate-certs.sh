#!/bin/bash
# scripts/generate-certs.sh
# Wrapper de inicialización

set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"

echo "Iniciando despliegue de certificados base..."

# Capturar IP si se pasa como argumento
[ -n "$1" ] && export SERVER_IP="$1"

"$DIR/scripts/renew-ca.sh"
"$DIR/scripts/renew-server-cert.sh"

echo ""
echo "✅ Todos los certificados están listos en 'certs/'."
echo "▶️ Servidor (VPS): server.crt / server.key"
echo "▶️ Cliente (Agente): ca.crt (Este debe ir a las raspberry pis/clientes)"
echo ""
echo "💡 Recuerda ejecutar 'scripts/setup-cron.sh' en el servidor para automatizar la renovación."
echo "=========================================="
