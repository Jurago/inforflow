# Inforflow

Análise de tráfego ISP em tempo real: **NetFlow v9** + **SNMP** + **BGP** do roteador de borda **170.245.127.191**.

## Fontes de dados

| Fonte | Endpoint | Uso |
|-------|----------|-----|
| NetFlow v9 | UDP `:2055` | Classificação CDN, Netflix, Globo, streaming, peers (ASN/next-hop/ifIndex) |
| SNMP v2c | `170.245.127.191:15161` community `infornetV2` | Interfaces (dedupe Eth-Trunk), Mbps, CPU, mem |
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

Auth opcional: defina `api_token` em `config.json` ou `INFORFLOW_API_TOKEN`. Enviei `X-API-Token` ou `?token=`.

## Configuração

Arquivo `config.json` (ou env `INFORFLOW_*`):

- `sampling_rate`: `0` = auto (prioriza amostragem nativa do template NetFlow; senão SNMP/NetFlow); ou fator fixo (ex. `1000`)
- `alert_util_pct`: limiar de alerta de utilização uplink
- `alert_asn_pct` / `alert_asn_mbps`: alerta quando um ASN de destino excede % do uplink ou Mbps absoluto
- `data_dir`: histórico, feeds, `asn_daily.json`, `sampling_native.json`
## Execução

```bash
./start.sh
```

UI: http://localhost:8080  
API: http://localhost:9090  
