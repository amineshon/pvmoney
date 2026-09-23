package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func Connect(dsn string) (*sql.DB, error) {
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)

	var last error
	for i := 0; i < 20; i++ {
		if err := database.Ping(); err == nil {
			if err := migrate(database); err != nil {
				return nil, fmt.Errorf("migrate: %w", err)
			}
			return database, nil
		} else {
			last = err
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil, fmt.Errorf("database unreachable: %w", last)
}

func migrate(database *sql.DB) error {
	_, err := database.Exec(`
CREATE TABLE IF NOT EXISTS accounts (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  bank_name TEXT NOT NULL DEFAULT '',
  account_number TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL DEFAULT 'bank',
  balance BIGINT NOT NULL DEFAULT 0,
  color TEXT NOT NULL DEFAULT '#c9a227',
  icon TEXT NOT NULL DEFAULT 'landmark',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  target_amount BIGINT NOT NULL DEFAULT 0,
  current_amount BIGINT NOT NULL DEFAULT 0,
  base_amount BIGINT NOT NULL DEFAULT 0,
  deadline DATE,
  color TEXT NOT NULL DEFAULT '#34d399',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS assets (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'other',
  quantity DOUBLE PRECISION NOT NULL DEFAULT 1,
  unit TEXT NOT NULL DEFAULT 'عدد',
  unit_value BIGINT NOT NULL DEFAULT 0,
  value BIGINT NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  color TEXT NOT NULL DEFAULT '#e0c36a',
  source_project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_items (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  planned_amount BIGINT NOT NULL DEFAULT 0,
  paid_amount BIGINT NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transactions (
  id UUID PRIMARY KEY,
  account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
  to_account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
  project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
  type TEXT NOT NULL,
  amount BIGINT NOT NULL,
  category TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE projects ADD COLUMN IF NOT EXISTS asset_id UUID;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS result_asset_type TEXT NOT NULL DEFAULT '';
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS item_id UUID;

CREATE INDEX IF NOT EXISTS idx_tx_occurred ON transactions (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_tx_account ON transactions (account_id);
CREATE INDEX IF NOT EXISTS idx_tx_project ON transactions (project_id);
CREATE INDEX IF NOT EXISTS idx_tx_type ON transactions (type);
CREATE INDEX IF NOT EXISTS idx_items_project ON project_items (project_id);
CREATE INDEX IF NOT EXISTS idx_assets_type ON assets (type);

CREATE TABLE IF NOT EXISTS debts (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'loan',
  creditor TEXT NOT NULL DEFAULT '',
  total_amount BIGINT NOT NULL DEFAULT 0,
  remaining BIGINT NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  color TEXT NOT NULL DEFAULT '#fb7185',
  has_schedule BOOLEAN NOT NULL DEFAULT FALSE,
  start_date DATE,
  end_date DATE,
  monthly_amount BIGINT NOT NULL DEFAULT 0,
  due_day INT NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS debt_installments (
  id UUID PRIMARY KEY,
  debt_id UUID NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
  amount BIGINT NOT NULL,
  due_date DATE NOT NULL,
  paid_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'pending',
  account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS debt_reminders (
  installment_id UUID NOT NULL REFERENCES debt_installments(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (installment_id, kind)
);

CREATE TABLE IF NOT EXISTS app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_inst_due ON debt_installments (due_date, status);

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  username TEXT NOT NULL,
  telegram_phone TEXT NOT NULL DEFAULT '',
  telegram_chat_id TEXT NOT NULL DEFAULT '',
  telegram_verified BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_login_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower ON users (LOWER(username));
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone ON users (telegram_phone) WHERE telegram_phone <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_chat ON users (telegram_chat_id) WHERE telegram_chat_id <> '';

CREATE TABLE IF NOT EXISTS sessions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions (user_id);

CREATE TABLE IF NOT EXISTS auth_challenges (
  id UUID PRIMARY KEY,
  purpose TEXT NOT NULL,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  username TEXT NOT NULL DEFAULT '',
  telegram_phone TEXT NOT NULL DEFAULT '',
  lang TEXT NOT NULL DEFAULT 'fa',
  link_token TEXT NOT NULL UNIQUE,
  otp_hash TEXT NOT NULL DEFAULT '',
  otp_expires_at TIMESTAMPTZ,
  chat_id TEXT NOT NULL DEFAULT '',
  phone_from_tg TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending_start',
  attempts INT NOT NULL DEFAULT 0,
  otp_sent_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_challenges_status ON auth_challenges (status, expires_at);
ALTER TABLE debts ADD COLUMN IF NOT EXISTS commission_amount BIGINT NOT NULL DEFAULT 0;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS prior_amount BIGINT NOT NULL DEFAULT 0;
ALTER TABLE project_items ADD COLUMN IF NOT EXISTS prior_amount BIGINT NOT NULL DEFAULT 0;
`)
	if err != nil {
		return err
	}
	_, err = database.Exec(`
ALTER TABLE projects ADD COLUMN IF NOT EXISTS base_amount BIGINT NOT NULL DEFAULT 0;
UPDATE projects SET base_amount = target_amount
WHERE NOT EXISTS (SELECT 1 FROM app_settings WHERE key = 'budget_model_v2');
INSERT INTO app_settings (key, value) VALUES ('budget_model_v2', '1')
ON CONFLICT (key) DO NOTHING;
UPDATE projects p SET target_amount = p.base_amount + COALESCE((
  SELECT SUM(planned_amount) FROM project_items i WHERE i.project_id = p.id
), 0);
UPDATE accounts SET balance = LEAST(GREATEST(balance, 0), 10000000000000)
WHERE balance < 0 OR balance > 10000000000000;
UPDATE transactions SET amount = LEAST(amount, 10000000000000)
WHERE amount > 10000000000000;
UPDATE assets SET unit_value = LEAST(unit_value, 10000000000000), value = LEAST(value, 10000000000000)
WHERE unit_value > 10000000000000 OR value > 10000000000000;
`)
	if err != nil {
		return err
	}
	if err := repairOrphanedAccountHistory(database); err != nil {
		return err
	}
	return rebuildLedgers(database)
}

func rebuildLedgers(database *sql.DB) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO app_settings (key, value) VALUES ('ledger_rebuild_v1', '1') ON CONFLICT (key) DO NOTHING`)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil
	}

	if _, err := tx.Exec(`
UPDATE debts SET
  total_amount = LEAST(GREATEST(total_amount, 0), 10000000000000),
  remaining = LEAST(GREATEST(remaining, 0), 10000000000000),
  monthly_amount = LEAST(GREATEST(monthly_amount, 0), 10000000000000),
  commission_amount = LEAST(GREATEST(commission_amount, 0), 10000000000000)`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
UPDATE debts d SET remaining = COALESCE((
  SELECT SUM(amount) FROM debt_installments i
  WHERE i.debt_id = d.id AND i.status = 'pending'
), 0)
WHERE has_schedule`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
UPDATE debts SET remaining = total_amount
WHERE total_amount > 0 AND remaining > total_amount`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
UPDATE accounts a SET
  balance = LEAST(GREATEST(COALESCE((
    SELECT SUM(
      CASE
        WHEN t.type IN ('opening','income','withdrawal') AND t.account_id = a.id THEN t.amount
        WHEN t.type = 'adjustment' AND t.account_id = a.id THEN t.amount
        WHEN t.type IN ('expense','contribution') AND t.account_id = a.id THEN -t.amount
        WHEN t.type = 'transfer' AND t.account_id = a.id THEN -t.amount
        WHEN t.type = 'transfer' AND t.to_account_id = a.id THEN t.amount
        ELSE 0
      END
    )
    FROM transactions t
    WHERE t.account_id = a.id OR t.to_account_id = a.id
  ), 0), 0), 10000000000000),
  updated_at = NOW()`); err != nil {
		return err
	}

	return tx.Commit()
}

// repairOrphanedAccountHistory undoes leftover transfers and deletes ghost
// transactions left behind when an account was deleted without reversing txs.
func repairOrphanedAccountHistory(database *sql.DB) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO app_settings (key, value) VALUES ('orphan_tx_repair_v1', '1') ON CONFLICT (key) DO NOTHING`)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil
	}

	if _, err := tx.Exec(`
UPDATE accounts a
SET balance = GREATEST(a.balance - x.total, 0), updated_at = NOW()
FROM (
  SELECT to_account_id AS id, SUM(amount) AS total
  FROM transactions
  WHERE type = 'transfer' AND account_id IS NULL AND to_account_id IS NOT NULL
  GROUP BY to_account_id
) x
WHERE a.id = x.id`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
UPDATE accounts a
SET balance = LEAST(a.balance + x.total, 10000000000000), updated_at = NOW()
FROM (
  SELECT account_id AS id, SUM(amount) AS total
  FROM transactions
  WHERE type = 'transfer' AND to_account_id IS NULL AND account_id IS NOT NULL
  GROUP BY account_id
) x
WHERE a.id = x.id`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
UPDATE projects p
SET current_amount = GREATEST(p.current_amount - x.total, 0), status = 'active', updated_at = NOW()
FROM (
  SELECT project_id AS id, SUM(amount) AS total
  FROM transactions
  WHERE type IN ('expense', 'contribution')
    AND account_id IS NULL AND to_account_id IS NULL AND project_id IS NOT NULL
  GROUP BY project_id
) x
WHERE p.id = x.id`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
UPDATE project_items i
SET paid_amount = GREATEST(i.paid_amount - x.total, 0), updated_at = NOW()
FROM (
  SELECT item_id AS id, SUM(amount) AS total
  FROM transactions
  WHERE item_id IS NOT NULL AND account_id IS NULL AND to_account_id IS NULL
  GROUP BY item_id
) x
WHERE i.id = x.id`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
DELETE FROM transactions
WHERE (account_id IS NULL AND to_account_id IS NULL)
   OR (type = 'transfer' AND account_id IS NULL AND to_account_id IS NOT NULL)
   OR (type = 'transfer' AND to_account_id IS NULL AND account_id IS NOT NULL)`); err != nil {
		return err
	}

	return tx.Commit()
}
