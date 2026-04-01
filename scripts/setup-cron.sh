#!/bin/bash
# scripts/setup-cron.sh
# Instala el cron e indaga sobre el comando de reinicio

set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"
RESTART_FILE="$DIR/scripts/.restart_cmd"
SCRIPT_TO_CRON="$DIR/scripts/renew-server-cert.sh"

echo "=========================================="
echo "🕒 Configuración de Renovación Automática (CRON)"
echo "=========================================="

if [ -f "$RESTART_FILE" ]; then
    current_cmd=$(cat "$RESTART_FILE")
fi

if [ -n "$current_cmd" ]; then
    read -p "El comando actual de reinicio es: '$current_cmd'. ¿Deseas cambiarlo? (y/n) [n]: " change_cmd
    change_cmd=${change_cmd:-n}
else
    echo "No existe un comando de reinicio configurado."
    change_cmd="y"
fi

if [[ "$change_cmd" =~ ^[Yy]$ ]]; then
    echo "--------------------------------------------------------"
    echo "Introduce el comando exacto de Linux para reiniciar el servidor de Netconnector."
    echo "Ejemplo: systemctl restart netconnector-server"
    echo "--------------------------------------------------------"
    read -p "Comando: " new_cmd
    if [ -n "$new_cmd" ]; then
        echo "$new_cmd" > "$RESTART_FILE"
        echo "✅ Comando de reinicio guardado en $RESTART_FILE"
    else
        echo "⚠️ No ingresaste ningún comando. No se reiniciará automáticamente el servicio tras renovar."
        echo "" > "$RESTART_FILE"
    fi
fi

# Hacer todo ejecutable por si acaso
chmod +x "$DIR/scripts/"*.sh

# Instalar cron
CRON_JOB="0 3 * * 0 $SCRIPT_TO_CRON >> $DIR/certs/renew.log 2>&1"

# Verificar si el cron ya existe
if crontab -l 2>/dev/null | grep -Fq "$SCRIPT_TO_CRON"; then
    CRON_EXISTS=true
else
    CRON_EXISTS=false
fi

if [ "$CRON_EXISTS" = true ]; then
    echo "✅ La regla en el cronjob ya se encuentra instalada para tu usuario."
else
    echo "Añadiendo rutina semanal al crontab del usuario..."
    (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -
    echo "✅ Cron configurado exitosamente. Se ejecutará cada domingo a las 3:00 AM."
fi

echo "=========================================="
