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

- `/` — Dashboard (resumo: KPIs, gap, sparkline 1h, BGP, ifaces, talkers; links para detalhes)
- `/graphs` — Séries temporais (1h–7d, IPv4/IPv6, ASN dest/peer, CDN/Streaming, zoom, export)
- `/router` `/router/detail?ifindex=` — SNMP por role, histórico uplink, filtros, detalhe de iface
- `/cdns` `/cdn/detail?name=Cloudflare` `/streaming` `/streaming/detail` `/peers` `/peers/detail` `/asn` `/asn/detail` — Detalhes por categoria / serviço / peer / ASN
- `/flows` — Explorador com filtro por IP, categoria e ASN (`?asn=AS15169`)
- `/sampling` `/cache` — Amostragem e cache

## API

| Rota | Descrição |
|------|-----------|
| `GET /api/dashboard` | Snapshot slim do dashboard (KPIs, gap, blocks, sparkline, BGP top, flows) |
| `GET /api/stats` | Agregados completos + SNMP + BGP + sampling + alerts + breakdowns |
| `GET /api/asn` | Snapshot slim da página ASN (destinos, peers, daily, names) |
| `GET /api/asn/daily` | Acumulado do dia por ASN de destino |
| `GET /api/asn/detail?asn=AS15169&hours=6` | Detalhe de um ASN (live, histórico, flows) |
| `GET /api/peers` | Snapshot Peers (BGP + Mbps scaled + downs + SNMP IX/transit + flows) |
| `GET /api/peers/detail?asn=AS26162&hours=6` | Detalhe de um peer ASN |
| `GET /api/cdn` | Snapshot CDN (rates+ASN, cache hit, % uplink, feeds, flows, overlap note) |
| `GET /api/cdn/detail?name=Cloudflare&hours=6` | Detalhe de um CDN (live, histórico, SNMP match, flows) |
| `GET /api/streaming` | Snapshot Streaming (rates, cache hit, % uplink, IPv4/IPv6, flows) |
| `GET /api/streaming/detail?name=YouTube&hours=6` | Detalhe de um serviço (live, histórico, SNMP match, flows) |
| `GET /api/router` | Snapshot roteador (SNMP + roles + alertas + NF por role; sem community) |
| `GET /api/router/detail?ifindex=12&hours=1` | Detalhe de iface (série SNMP, flows in/out if) |
| `GET /api/snmp` | Snapshot SNMP bruto |
| `GET /api/bgp` | Sessões BGP |
| `GET /api/flows?asn=&ip=&category=&q=` | Flows recentes (filtros; ring ~2000 + índice por ASN) |
| `GET /api/history?hours=24&max_points=800&from=&to=` | Histórico (`by_*` + IPv4/IPv6 + `by_snmp_role_*`) |
| `GET /api/history/compare?hours=24` | Compara período atual vs anterior |
| `GET /api/alerts` | Alertas ativos (utilização, BGP, ASN alto, etc.) |
| `GET /api/sampling` | Fator NetFlow×SNMP (nativo/auto/fixo) |
| `GET /api/talkers` | Top clientes CGNAT `100.64.x` |
| `GET /api/ipapi` | Status do resolvedor ASN (ip-api.com) |
| `GET /api/export?kind=cdn\|streaming\|peers\|asn\|flows\|talkers\|router\|history&format=csv\|json` | Export |
| `GET /api/health` | Saúde (sem auth) |

Config relevante (`config.json`):
- `alert_asn_ignore` — ASNs ignorados em alertas `asn_high_*` (Google/Meta/CF/…)
- `asn_history_top` — top-N por amostra no histórico (padrão 30)
- `asn_watched` — ASNs sempre mantidos no histórico
- `cdn_watched` — CDNs sempre mantidos no histórico (padrão: Cloudflare, Akamai, Google Cache, AWS CloudFront, Fastly)
- `asn_digest_hour` — hora local do digest diário via webhook/telegram (`-1` desliga)
- `ix_asn` — ASN do IX usado no KPI da página Peers (padrão 26162 IX.br)

Página `/asn/detail?asn=AS…` — série, peer e flows indexados por ASN.
Página `/peers/detail?asn=AS…` — sessões BGP, histórico e flows do peer.

Nomes de ASN: resolução via **ip-api.com** (cache em `data/ipapi_asn.json`). Destino e peer BGP ficam em listas separadas.

Alertas externos (opcional em `config.json`): `webhook_url`, `telegram_bot_token` + `telegram_chat_id`, `alert_util_pct`, `alert_asn_pct` / `alert_asn_mbps`.

Auth: `INFORFLOW_API_TOKEN`, `INFORFLOW_UI_USER` e `INFORFLOW_UI_PASSWORD` em `/etc/inforflow/secrets.env`. Login em `/login` gera token de sessão (24h). API usa header `X-API-Token`.

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
