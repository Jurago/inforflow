#!/bin/bash
# Healthcheck Inforflow — systemd timer + UptimeRobot
set -e
ROOT="${INFORFLOW_ROOT:-/var/www/html/inforflow}"
API="${INFORFLOW_HEALTH_URL:-http://127.0.0.1:9090/api/health}"
TOKEN="${INFORFLOW_API_TOKEN:-}"
HDR=()
[ -n "$TOKEN" ] && HDR=(-H "X-API-Token: $TOKEN")

# Assets críticos da UI (evita dashboard mudo)
for f in \
  "$ROOT/static/js/inforflow.js" \
  "$ROOT/static/js/dashboard.js" \
  "$ROOT/static/js/auth.js" \
  "$ROOT/static/css/inforflow.css" \
  "$ROOT/bin/inforflow" \
  "$ROOT/bin/inforflow-collector"
do
  if [ ! -s "$f" ]; then
    echo "FAIL missing asset $f"
    exit 1
  fi
done

# Referências de script no layout Rust devem existir em disco
missing=0
while IFS= read -r js; do
  [ -z "$js" ] && continue
  if [ ! -s "$ROOT/static/js/$js" ]; then
    echo "FAIL layout references missing static/js/$js"
    missing=1
  fi
done < <(grep -oE 'static/js/[a-z0-9_]+\.js' "$ROOT/server/src/pages/layout.rs" 2>/dev/null | sed 's|static/js/||' | sort -u || true)
[ "$missing" = "0" ] || exit 1

code=$(curl -sf "${HDR[@]}" -o /tmp/inforflow-health.json -w '%{http_code}' "$API" || echo 000)
if [ "$code" != "200" ]; then
  echo "FAIL health HTTP $code"
  exit 1
fi

# API autenticada (se token disponível)
if [ -n "$TOKEN" ]; then
  dcode=$(curl -sf -H "X-API-Token: $TOKEN" -o /tmp/inforflow-dash.json -w '%{http_code}' \
    http://127.0.0.1:9090/api/dashboard || echo 000)
  if [ "$dcode" != "200" ]; then
    echo "FAIL dashboard API HTTP $dcode"
    exit 1
  fi
fi

# Static via nginx/local file size check already done; optional HTTP probe
if curl -sf -o /dev/null -m 3 http://127.0.0.1:8080/static/js/inforflow.js 2>/dev/null; then
  :
elif [ -s "$ROOT/static/js/inforflow.js" ]; then
  :
else
  echo "FAIL inforflow.js unreachable"
  exit 1
fi

python3 - <<'PY'
import json, sys, shutil
d=json.load(open("/tmp/inforflow-health.json"))
if d.get("status")!="ok":
    print("FAIL status", d.get("status")); sys.exit(1)
if not d.get("snmp"):
    print("FAIL snmp offline"); sys.exit(1)
silent = int(d.get("netflow_silent_s") or 0)
snmp_mbps = float(d.get("snmp_mbps") or 0)
udp_queue = int(d.get("udp_queue") or 0)
udp_rcvbuf = int(d.get("udp_rcvbuf") or 0)
# Silêncio com uplink ativo
if silent > 90 and snmp_mbps > 50:
    print(f"FAIL netflow silent {silent}s with uplink {snmp_mbps:.0f} Mbps"); sys.exit(1)
# Fila UDP perigosamente alta (>50% do rcvbuf ou >16MB)
lim = max(16*1024*1024, int(udp_rcvbuf * 0.5) if udp_rcvbuf else 16*1024*1024)
if udp_queue > lim:
    print(f"FAIL udp_queue {udp_queue} > {lim}"); sys.exit(1)
du = shutil.disk_usage("/var/www/html/inforflow/data")
if du.free < 500*1024*1024:
    print(f"FAIL disk low free={du.free//1024//1024}MB"); sys.exit(1)
s3 = d.get("s3") or {}
s3note = "s3=off"
if s3.get("enabled"):
    age = s3.get("age_s")
    err = s3.get("last_error") or ""
    s3note = f"s3=on age={age if age is not None else 'pending'}s"
    if err:
        s3note += f" err={err[:60]}"
print("OK", round(d.get("mbps_scaled",0),1), "Mbps", d.get("flows"), "flows",
      "bgp", d.get("bgp_peers"), "silent", silent, "s",
      "udp_q", udp_queue, "workers", d.get("ingest_workers"), s3note)
PY

# Heartbeat Uptime Kuma (monitor tipo Push)
if [ -f /etc/inforflow/kuma-push.env ]; then
  # shellcheck disable=SC1091
  . /etc/inforflow/kuma-push.env
fi
if [ -n "${INFORFLOW_UPTIMEKUMA_PUSH_URL:-}" ]; then
  curl -sf -m 10 "${INFORFLOW_UPTIMEKUMA_PUSH_URL}?status=up&msg=OK&ping=0" >/dev/null \
    || echo "WARN push Uptime Kuma falhou"
fi
