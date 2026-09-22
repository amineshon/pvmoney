# استقرار روی pvmoney.gowin.ir

مسیر پیشنهادی روی سرور: `/opt/pvmoney`

پورت‌ها (با بقیه سایت‌های gowin تداخل ندارد):

| سرویس | پورت localhost |
|--------|----------------|
| frontend | `127.0.0.1:3011` |
| backend | `127.0.0.1:8083` |
| postgres | فقط داخل داکر |

## ۱. DNS

رکورد A:

```
pvmoney.gowin.ir  →  IP سرور
```

## ۲. کپی پروژه

از ویندوز (PowerShell)، پوشه پروژه را به سرور بفرست. `node_modules` و `.next` و باینری `backend/server` لازم نیست:

```powershell
scp -r D:\project\td\pvmoney user@SERVER:/opt/pvmoney
```

یا روی خود سرور:

```bash
sudo mkdir -p /opt/pvmoney
sudo chown "$USER:$USER" /opt/pvmoney
# بعد rsync / git / scp
```

## ۳. env

```bash
cd /opt/pvmoney
cp .env.example .env
nano .env
```

حداقل این‌ها را پر کن:

```
TELEGRAM_BOT_TOKEN=...
POSTGRES_PASSWORD=یک‌رمز-قوی
CORS_ORIGIN=https://pvmoney.gowin.ir
```

## ۴. بالا آوردن

```bash
chmod +x deploy/up.sh
./deploy/up.sh
```

یا دستی:

```bash
docker compose -f docker-compose.prod.yml up -d --build
sudo cp deploy/nginx-pvmoney.gowin.ir.conf /etc/nginx/sites-available/pvmoney.gowin.ir
sudo ln -sf /etc/nginx/sites-available/pvmoney.gowin.ir /etc/nginx/sites-enabled/pvmoney.gowin.ir
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d pvmoney.gowin.ir
```

## ۵. چک

```bash
curl -s http://127.0.0.1:8083/health
curl -sI http://pvmoney.gowin.ir
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f --tail=80
```

سایت باید روی `https://pvmoney.gowin.ir` باز شود. ثبت‌نام همان جریان تلگرام است.

## به‌روزرسانی بعدی

```bash
cd /opt/pvmoney
# فایل‌های جدید را کپی کن
docker compose -f docker-compose.prod.yml up -d --build
```

دیتابیس روی volume `pgdata` می‌ماند.
