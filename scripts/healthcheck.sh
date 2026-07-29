#!/bin/bash
# Healthcheck Inforflow — systemd timer + UptimeRobot
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
import json, sys, shutil
d=json.load(open("/tmp/inforflow-health.json"))
if d.get("status")!="ok":
    print("FAIL status", d.get("status")); sys.exit(1)
if not d.get("snmp"):
    print("FAIL snmp offline"); sys.exit(1)
silent = int(d.get("netflow_silent_s") or 0)
snmp_mbps = float(d.get("snmp_mbps") or 0)
if silent > 120 and snmp_mbps > 50:
    print(f"FAIL netflow silent {silent}s with uplink {snmp_mbps:.0f} Mbps"); sys.exit(1)
du = shutil.disk_usage("/var/www/html/inforflow/data")
if du.free < 500*1024*1024:
    print(f"FAIL disk low free={du.free//1024//1024}MB"); sys.exit(1)
print("OK", round(d.get("mbps_scaled",0),1), "Mbps", d.get("flows"), "flows",
      "bgp", d.get("bgp_peers"), "silent", silent, "s")
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
