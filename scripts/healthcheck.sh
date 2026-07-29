#!/bin/bash
# Healthcheck Inforflow — usado pelo systemd e cron
set -e
API="${INFORFLOW_HEALTH_URL:-http://127.0.0.1:9090/api/health}"
TOKEN="${INFORFLOW_API_TOKEN:-}"
HDR=()
[ -n "$TOKEN" ] && HDR=(-H "X-API-Token: $TOKEN")
code=$(curl -sf "${HDR[@]}" -o /tmp/inforflow-health.json -w '%{http_code}' "$API" || echo 000)
if [ "$code" != "200" ]; then
  echo "FAIL health HTTP $code"
  exit 1
fi
python3 - <<'PY'
import json, sys, time
d=json.load(open("/tmp/inforflow-health.json"))
if d.get("status")!="ok":
    sys.exit(1)
# NetFlow silencioso com uplink ativo seria alertado pelo coletor
print("OK", d.get("mbps_scaled"), "Mbps scaled", d.get("flows"), "flows")
PY
