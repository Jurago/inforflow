#!/usr/bin/env python3
"""Configura monitores Inforflow no Uptime Kuma (Socket.IO + verificação API key)."""
from __future__ import annotations

import os
import sys
import urllib.request
import base64

KUMA_URL = os.environ.get("INFORFLOW_UPTIMEKUMA_URL", "http://170.245.127.16:3001").rstrip("/")
KUMA_API_KEY = os.environ.get("INFORFLOW_UPTIMEKUMA_API_KEY", "")
KUMA_USER = os.environ.get("INFORFLOW_UPTIMEKUMA_USER", "")
KUMA_PASS = os.environ.get("INFORFLOW_UPTIMEKUMA_PASSWORD", "")
PUBLIC_URL = os.environ.get("INFORFLOW_PUBLIC_URL", "https://inforflow.infornetmg.com.br").rstrip("/")
PUSH_ENV = os.environ.get("INFORFLOW_UPTIMEKUMA_PUSH_FILE", "/etc/inforflow/kuma-push.env")

MONITORS = [
    {
        "name": "Inforflow Health API",
        "type": "http",
        "url": f"{PUBLIC_URL}/api/health",
        "interval": 60,
        "maxretries": 2,
        "accepted_statuscodes": ["200-299"],
    },
    {
        "name": "Inforflow Dashboard",
        "type": "http",
        "url": f"{PUBLIC_URL}/",
        "interval": 120,
        "maxretries": 2,
        "accepted_statuscodes": ["200-299"],
    },
    {
        "name": "Inforflow Login",
        "type": "http",
        "url": f"{PUBLIC_URL}/login",
        "interval": 300,
        "maxretries": 2,
        "accepted_statuscodes": ["200-299"],
    },
    {
        "name": "Inforflow Collector Push",
        "type": "push",
        "interval": 120,
        "maxretries": 1,
        "heartbeatInterval": 180,
    },
]


def verify_api_key() -> bool:
    if not KUMA_API_KEY:
        print("AVISO: INFORFLOW_UPTIMEKUMA_API_KEY não definida")
        return False
    auth = base64.b64encode(f":{KUMA_API_KEY}".encode()).decode()
    req = urllib.request.Request(
        f"{KUMA_URL}/metrics",
        headers={"Authorization": f"Basic {auth}"},
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            body = resp.read(200).decode(errors="replace")
            print(f"OK API key — /metrics responde ({len(body)}+ bytes)")
            return True
    except Exception as e:
        print(f"FALHA /metrics: {e}")
        return False


def setup_monitors() -> int:
    if not KUMA_USER or not KUMA_PASS:
        print("\nPara criar monitores automaticamente, defina em /etc/inforflow/secrets.env:")
        print("  INFORFLOW_UPTIMEKUMA_USER=<usuário admin do Kuma>")
        print("  INFORFLOW_UPTIMEKUMA_PASSWORD=<senha>")
        print("\nMonitores sugeridos (criar manualmente no Uptime Kuma):")
        for m in MONITORS:
            if m["type"] == "http":
                print(f"  • HTTP: {m['name']} → {m['url']} (intervalo {m['interval']}s)")
            else:
                print(f"  • Push: {m['name']} (intervalo heartbeat {m.get('heartbeatInterval', 180)}s)")
        return 0

    try:
        from uptime_kuma_api import UptimeKumaApi, MonitorType
    except ImportError:
        print("Instale: python3 -m venv /opt/inforflow-kuma && pip install uptime-kuma-api")
        return 1

    api = UptimeKumaApi(KUMA_URL)
    try:
        res = api.login(KUMA_USER, KUMA_PASS)
        if not res.get("ok", True):
            print("Login falhou:", res)
            return 1
        print("Login OK como", KUMA_USER)

        existing = {m.get("name"): m for m in api.get_monitors()}
        push_url = None

        for spec in MONITORS:
            if spec["name"] in existing:
                print("  já existe:", spec["name"], "(id", existing[spec["name"]].get("id"), ")")
                if spec["type"] == "push":
                    tok = existing[spec["name"]].get("pushToken")
                    if tok:
                        push_url = f"{KUMA_URL}/api/push/{tok}"
                continue

            if spec["type"] == "http":
                r = api.add_monitor(
                    type=MonitorType.HTTP,
                    name=spec["name"],
                    url=spec["url"],
                    interval=spec["interval"],
                    maxretries=spec["maxretries"],
                    accepted_statuscodes=spec["accepted_statuscodes"],
                )
            else:
                r = api.add_monitor(
                    type=MonitorType.PUSH,
                    name=spec["name"],
                    interval=spec["interval"],
                    maxretries=spec["maxretries"],
                    heartbeatInterval=spec.get("heartbeatInterval", 180),
                )
                mid = r.get("monitorID")
                if mid:
                    detail = api.get_monitor(mid)
                    tok = detail.get("pushToken")
                    if tok:
                        push_url = f"{KUMA_URL}/api/push/{tok}"
            print("  criado:", spec["name"], "→", r.get("msg", r))

        if push_url:
            with open(PUSH_ENV, "w") as f:
                f.write(f"INFORFLOW_UPTIMEKUMA_PUSH_URL={push_url}\n")
            os.chmod(PUSH_ENV, 0o600)
            print(f"\nPush URL salva em {PUSH_ENV}")
            print("O healthcheck enviará heartbeat a cada execução.")
        return 0
    except Exception as e:
        print("Erro:", e)
        return 1
    finally:
        api.disconnect()


def main() -> int:
    print("=== Uptime Kuma — Inforflow ===")
    print("URL:", KUMA_URL)
    verify_api_key()
    return setup_monitors()


if __name__ == "__main__":
    sys.exit(main())
