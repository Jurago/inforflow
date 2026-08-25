#!/bin/bash
# Emite certificado Let's Encrypt para inforflow.infornetmg.com.br
set -euo pipefail

DIR="$(cd "$(dirname "$0")/.." && pwd)"
DOMAIN="${INFORFLOW_SSL_DOMAIN:-inforflow.infornetmg.com.br}"
EMAIL="${INFORFLOW_SSL_EMAIL:-admin@infornetmg.com.br}"
SECRETS="${INFORFLOW_SECRETS_FILE:-/etc/inforflow/secrets.env}"
CF_INI="${INFORFLOW_CLOUDFLARE_INI:-/etc/inforflow/cloudflare.ini}"
LE_DIR="/etc/letsencrypt/live/${DOMAIN}"

if [ -f "$SECRETS" ]; then
  set -a
  # shellcheck disable=SC1090
  source "$SECRETS"
  set +a
fi

echo "=== SSL — ${DOMAIN} ==="

if [ -f "${LE_DIR}/fullchain.pem" ]; then
  echo "Certificado LE já existe em ${LE_DIR}"
else
  issued=0
  if [ -n "${INFORFLOW_CLOUDFLARE_API_TOKEN:-}" ]; then
    umask 077
    mkdir -p "$(dirname "$CF_INI")"
    printf 'dns_cloudflare_api_token = %s\n' "$INFORFLOW_CLOUDFLARE_API_TOKEN" >"$CF_INI"
    chmod 600 "$CF_INI"
    echo "Tentando desafio DNS (Cloudflare)..."
    if certbot certonly \
      --dns-cloudflare \
      --dns-cloudflare-credentials "$CF_INI" \
      -d "$DOMAIN" \
      --non-interactive \
      --agree-tos \
      -m "$EMAIL"; then
      issued=1
    else
      echo "AVISO: desafio DNS falhou"
    fi
  fi

  if [ "$issued" -eq 0 ]; then
    echo "Tentando desafio HTTP (nginx)..."
    certbot certonly \
      --nginx \
      -d "$DOMAIN" \
      --non-interactive \
      --agree-tos \
      -m "$EMAIL"
  fi
fi

if [ ! -f "${LE_DIR}/fullchain.pem" ]; then
  echo "FALHA: certificado não emitido"
  exit 1
fi

python3 - <<PY
from pathlib import Path

domain = "${DOMAIN}"
paths = {
    "/etc/nginx/ssl/inforflow.crt": f"/etc/letsencrypt/live/{domain}/fullchain.pem",
    "/etc/nginx/ssl/inforflow.key": f"/etc/letsencrypt/live/{domain}/privkey.pem",
}
for src, dst in (
    ("${DIR}/deploy/nginx-inforflow.conf", "/etc/nginx/sites-available/inforflow"),
):
    text = Path(src).read_text()
    for old, new in paths.items():
        text = text.replace(old, new)
    if "options-ssl-nginx.conf" not in text:
        text = text.replace(
            "ssl_protocols       TLSv1.2 TLSv1.3;",
            "include /etc/letsencrypt/options-ssl-nginx.conf;\n    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;",
        )
    Path(dst).write_text(text)
PY

ln -sf /etc/nginx/sites-available/inforflow /etc/nginx/sites-enabled/inforflow
nginx -t
systemctl reload nginx

echo "OK — certificado instalado:"
openssl x509 -in "${LE_DIR}/fullchain.pem" -noout -subject -issuer -dates
