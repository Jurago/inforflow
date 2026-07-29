# Inforflow

Análise de tráfego ISP em tempo real: **NetFlow v9** + **SNMP** + **BGP** do roteador de borda **170.245.127.191**.

## Fontes de dados

| Fonte | Endpoint | Uso |
|-------|----------|-----|
| NetFlow v9 | UDP `:2055` | Classificação CDN, Netflix, Globo, streaming, peers (ASN/next-hop/ifIndex) |
| SNMP v2c | `170.245.127.191:15161` | Interfaces (dedupe Eth-Trunk), Mbps, CPU, mem — community em `/etc/inforflow/secrets.env` |
| BGP4-MIB | via SNMP | Sessões BGP e atribuição de tráfego por ASN |
| Feeds | Cloudflare IPs (+ `data/feeds/extra.txt`) | Prefixos CDN atualizados periodicamente |

## Páginas

- `/` — Dashboard (NetFlow, SNMP, amostragem estimada, alertas)
- `/graphs` — Séries temporais com filtro 1h/6h/24h + comparar períodos
- `/router` — Interfaces SNMP
- `/cdns` `/streaming` `/peers` `/asn` — Detalhes por categoria / ASN de destino
- `/flows` — Explorador com filtro por IP, categoria e ASN (`?asn=AS15169`)
- `/sampling` `/cache` — Amostragem e cache

## API

| Rota | Descrição |
|------|-----------|
| `GET /api/stats` | Agregados + SNMP + BGP + sampling + alerts + `asn_breakdown` |
| `GET /api/snmp` | Snapshot SNMP |
| `GET /api/bgp` | Sessões BGP |
| `GET /api/flows?asn=&ip=&category=&q=` | Flows recentes (filtros) |
| `GET /api/history?hours=24` | Histórico (inclui `by_asn_mbps_scaled`) |
| `GET /api/history/compare?hours=24` | Compara período atual vs anterior |
| `GET /api/alerts` | Alertas ativos (utilização, BGP, ASN alto, etc.) |
| `GET /api/sampling` | Fator NetFlow×SNMP (nativo/auto/fixo) |
| `GET /api/talkers` | Top clientes CGNAT `100.64.x` |
| `GET /api/ipapi` | Status do resolvedor ASN (ip-api.com) |
| `GET /api/export?kind=cdn\|streaming\|peers\|asn\|flows\|talkers&format=csv\|json` | Export |
| `GET /api/health` | Saúde (sem auth) |

Nomes de ASN: resolução automática via **ip-api.com** (gratuita, batch) com cache em `data/ipapi_asn.json`. Peers BGP locais (ex.: AS269096 N&K Tecnologia) entram no breakdown ASN mesmo quando o DstAS do flow é o destino final.

Alertas externos (opcional em `config.json`):
- `webhook_url` — POST JSON
- `telegram_bot_token` + `telegram_chat_id`
- `alert_util_pct` — limiar de uplink
- `alert_asn_pct` / `alert_asn_mbps` — limiar por ASN de destino

Auth: defina `INFORFLOW_API_TOKEN`, `INFORFLOW_UI_USER` e `INFORFLOW_UI_PASSWORD` em `/etc/inforflow/secrets.env`. Login em `/login` gera **token de sessão** (24h). API usa header `X-API-Token`.

## Deploy (produção)

```bash
# 1. Secrets (chmod 600)
cp secrets.env.example /etc/inforflow/secrets.env
# editar: INFORFLOW_API_TOKEN, INFORFLOW_SNMP_COMMUNITY, UI user/pass, S3…

# 2. Build + systemd + nginx + healthcheck timer
./start.sh

# 3. UptimeRobot (monitores HTTP)
INFORFLOW_UPTIMEROBOT_API_KEY=... ./scripts/setup-uptimerobot.sh

# 4. Uptime Kuma (alertas + push heartbeat)
INFORFLOW_UPTIMEKUMA_API_KEY=uk2_... \
INFORFLOW_UPTIMEKUMA_USER=admin INFORFLOW_UPTIMEKUMA_PASSWORD=... \
python3 scripts/setup-uptimekuma.py

# 5. Verificar
./scripts/healthcheck.sh
journalctl -u inforflow-collector -f
```

Serviços systemd: `inforflow-collector`, `inforflow-web`, `inforflow-healthcheck.timer` (a cada 2 min).

API do coletor escuta em `127.0.0.1:9090` — exposta via nginx (`/api/`).

## Execução (dev)

```bash
./start.sh
```

UI: http://localhost:8080  
API: http://127.0.0.1:9090  
