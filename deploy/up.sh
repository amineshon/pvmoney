#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  echo "فایل .env نیست. اول این را بزن:"
  echo "  cp .env.example .env"
  echo "بعد TELEGRAM_BOT_TOKEN و POSTGRES_PASSWORD را پر کن."
  exit 1
fi

if grep -q 'TELEGRAM_BOT_TOKEN=$' .env || grep -q 'TELEGRAM_BOT_TOKEN=\s*$' .env; then
  echo "هشدار: TELEGRAM_BOT_TOKEN خالی است. ربات تلگرام کار نمی‌کند."
fi

echo "==> build & up"
docker compose -f docker-compose.prod.yml up -d --build

echo "==> nginx"
if [[ -d /etc/nginx/sites-available ]]; then
  sudo cp "$ROOT/deploy/nginx-pvmoney.gowin.ir.conf" /etc/nginx/sites-available/pvmoney.gowin.ir
  sudo ln -sf /etc/nginx/sites-available/pvmoney.gowin.ir /etc/nginx/sites-enabled/pvmoney.gowin.ir
  sudo nginx -t
  sudo systemctl reload nginx
  echo "nginx reload شد. برای SSL:"
  echo "  sudo certbot --nginx -d pvmoney.gowin.ir"
else
  echo "nginx sites-available پیدا نشد. فایل را دستی کپی کن:"
  echo "  $ROOT/deploy/nginx-pvmoney.gowin.ir.conf"
fi

echo
echo "فرانت: 127.0.0.1:3011"
echo "API:    127.0.0.1:8083/health"
echo "دامنه:  http://pvmoney.gowin.ir"
