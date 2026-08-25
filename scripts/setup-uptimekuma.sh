#!/bin/bash
# Wrapper — usa venv com uptime-kuma-api
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
VENV="${INFORFLOW_KUMA_VENV:-/opt/inforflow-kuma-venv}"
if [ ! -x "$VENV/bin/python3" ]; then
  python3 -m venv "$VENV"
  "$VENV/bin/pip" install -q uptime-kuma-api
fi
set -a
[ -f /etc/inforflow/secrets.env ] && . /etc/inforflow/secrets.env
set +a
"$VENV/bin/pip" install -q uptime-kuma-api python-socketio 2>/dev/null || true
exec "$VENV/bin/python3" "$DIR/setup-uptimekuma.py" "$@"
