#!/bin/bash
# Configura monitores UptimeRobot para Inforflow
set -e
API_KEY="${INFORFLOW_UPTIMEROBOT_API_KEY:-${1:-}}"
BASE="${INFORFLOW_PUBLIC_URL:-https://inforflow.infornetmg.com.br}"
if [ -z "$API_KEY" ]; then
  echo "Uso: INFORFLOW_UPTIMEROBOT_API_KEY=... $0"
  exit 1
fi

create_monitor() {
  local name="$1" url="$2" type="${3:-1}"
  curl -sf -X POST https://api.uptimerobot.com/v2/newMonitor \
    -d "api_key=${API_KEY}&format=json&friendly_name=${name}&url=${url}&type=${type}&interval=300" \
    | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('stat','?'), d.get('monitor',{}).get('id',''), d.get('error',{}).get('message',''))"
}

echo "=== Monitores UptimeRobot ==="
create_monitor "Inforflow Health API" "${BASE}/api/health"
create_monitor "Inforflow Dashboard" "${BASE}/"
create_monitor "Inforflow Login" "${BASE}/login"

echo "Monitores existentes:"
curl -sf -X POST https://api.uptimerobot.com/v2/getMonitors \
  -d "api_key=${API_KEY}&format=json&logs=0" \
  | python3 -c "
import sys,json
d=json.load(sys.stdin)
for m in d.get('monitors') or []:
    print(m.get('id'), m.get('friendly_name'), m.get('url'), 'status', m.get('status'))
"
