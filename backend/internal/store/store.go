package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pvmoney/internal/models"
	"pvmoney/internal/rates"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInsufficient  = errors.New("insufficient balance")
	ErrInvalid       = errors.New("invalid request")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrConflict      = errors.New("conflict")
	ErrTooMany       = errors.New("too_many_attempts")
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListAccounts(ctx context.Context) ([]models.Account, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, bank_name, account_number, type, balance, color, icon, created_at, updated_at
FROM accounts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Account, 0)
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAccount(ctx context.Context, id string) (models.Account, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, bank_name, account_number, type, balance, color, icon, created_at, updated_at
FROM accounts WHERE id = $1`, id)
	a, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Account{}, ErrNotFound
	}
	return a, err
}

func (s *Store) CreateAccount(ctx context.Context, in models.AccountInput) (models.Account, error) {
	if strings.TrimSpace(in.Name) == "" {
		return models.Account{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if in.Type == "" {
		in.Type = "bank"
	}
	if in.Color == "" {
		in.Color = "#c9a227"
	}
	if in.Icon == "" {
		in.Icon = "landmark"
	}
	if err := CheckMoney(in.Balance); err != nil {
		return models.Account{}, err
	}
	id := uuid.NewString()
	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Account{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
INSERT INTO accounts (id, name, bank_name, account_number, type, balance, color, icon, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`,
		id, strings.TrimSpace(in.Name), in.BankName, in.AccountNumber, in.Type, in.Balance, in.Color, in.Icon, now)
	if err != nil {
		return models.Account{}, err
	}
	if in.Balance != 0 {
		_, err = tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,'opening',$3,'موجودی اولیه','موجودی اولیه حساب',$4,$4)`,
			uuid.NewString(), id, in.Balance, now)
		if err != nil {
			return models.Account{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Account{}, err
	}
	return s.GetAccount(ctx, id)
}

func (s *Store) UpdateAccount(ctx context.Context, id string, in models.AccountInput) (models.Account, error) {
	res, err := s.db.ExecContext(ctx, `
UPDATE accounts SET name=$2, bank_name=$3, account_number=$4, type=$5, color=$6, icon=$7, updated_at=NOW()
WHERE id=$1`, id, strings.TrimSpace(in.Name), in.BankName, in.AccountNumber, in.Type, in.Color, in.Icon)
	if err != nil {
		return models.Account{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.Account{}, ErrNotFound
	}
	return s.GetAccount(ctx, id)
}

func (s *Store) DeleteAccount(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists string
	err = tx.QueryRowContext(ctx, `SELECT id FROM accounts WHERE id=$1 FOR UPDATE`, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	// Transfers out of this account: take the leftover credit back from destinations.
	if _, err := tx.ExecContext(ctx, `
UPDATE accounts a
SET balance = GREATEST(a.balance - x.total, 0), updated_at = NOW()
FROM (
  SELECT to_account_id AS id, SUM(amount) AS total
  FROM transactions
  WHERE type = 'transfer' AND account_id = $1 AND to_account_id IS NOT NULL AND to_account_id <> $1
  GROUP BY to_account_id
) x
WHERE a.id = x.id`, id); err != nil {
		return err
	}

	// Transfers into this account: give the money back to the source accounts.
	if _, err := tx.ExecContext(ctx, `
UPDATE accounts a
SET balance = LEAST(a.balance + x.total, $2), updated_at = NOW()
FROM (
  SELECT account_id AS id, SUM(amount) AS total
  FROM transactions
  WHERE type = 'transfer' AND to_account_id = $1 AND account_id IS NOT NULL AND account_id <> $1
  GROUP BY account_id
) x
WHERE a.id = x.id`, id, models.MaxMoney); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE projects p
SET current_amount = GREATEST(p.current_amount - x.total, 0), status = 'active', updated_at = NOW()
FROM (
  SELECT project_id AS id, SUM(amount) AS total
  FROM transactions
  WHERE account_id = $1 AND project_id IS NOT NULL AND type IN ('expense', 'contribution')
  GROUP BY project_id
) x
WHERE p.id = x.id`, id); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE project_items i
SET paid_amount = GREATEST(i.paid_amount - x.total, 0), updated_at = NOW()
FROM (
  SELECT item_id AS id, SUM(amount) AS total
  FROM transactions
  WHERE account_id = $1 AND item_id IS NOT NULL
  GROUP BY item_id
) x
WHERE i.id = x.id`, id); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE projects p
SET current_amount = p.current_amount + x.total, updated_at = NOW()
FROM (
  SELECT project_id AS id, SUM(amount) AS total
  FROM transactions
  WHERE account_id = $1 AND project_id IS NOT NULL AND type = 'withdrawal'
  GROUP BY project_id
) x
WHERE p.id = x.id`, id); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM transactions WHERE account_id = $1 OR to_account_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM accounts WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) AdjustAccount(ctx context.Context, id string, in models.AdjustInput) (models.Account, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Account{}, err
	}
	defer tx.Rollback()

	var current int64
	err = tx.QueryRowContext(ctx, `SELECT balance FROM accounts WHERE id=$1 FOR UPDATE`, id).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Account{}, ErrNotFound
	}
	if err != nil {
		return models.Account{}, err
	}
	if err := CheckMoney(in.Balance); err != nil {
		return models.Account{}, err
	}
	diff := in.Balance - current
	if diff == 0 {
		return s.GetAccount(ctx, id)
	}
	_, err = tx.ExecContext(ctx, `UPDATE accounts SET balance=$2, updated_at=NOW() WHERE id=$1`, id, in.Balance)
	if err != nil {
		return models.Account{}, err
	}
	desc := in.Description
	if desc == "" {
		desc = "تنظیم موجودی"
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,'adjustment',$3,'تنظیم',$4,NOW(),NOW())`, uuid.NewString(), id, diff, desc)
	if err != nil {
		return models.Account{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Account{}, err
	}
	return s.GetAccount(ctx, id)
}

func (s *Store) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, description, target_amount, current_amount, base_amount, deadline, color, status, created_at, updated_at, asset_id, result_asset_type
FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Project, 0)
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.attachProjectItems(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) attachProjectItems(ctx context.Context, projects []models.Project) error {
	if len(projects) == 0 {
		return nil
	}
	ids := make([]string, len(projects))
	idx := make(map[string]int, len(projects))
	for i, p := range projects {
		ids[i] = p.ID
		idx[p.ID] = i
		projects[i].Items = []models.ProjectItem{}
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_id, name, planned_amount, paid_amount, notes, created_at, updated_at
FROM project_items WHERE project_id = ANY($1) ORDER BY created_at`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var it models.ProjectItem
		if err := rows.Scan(&it.ID, &it.ProjectID, &it.Name, &it.PlannedAmount, &it.PaidAmount, &it.Notes, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return err
		}
		i, ok := idx[it.ProjectID]
		if !ok {
			continue
		}
		projects[i].Items = append(projects[i].Items, it)
	}
	return rows.Err()
}

func (s *Store) GetProject(ctx context.Context, id string) (models.Project, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, description, target_amount, current_amount, base_amount, deadline, color, status, created_at, updated_at, asset_id, result_asset_type
FROM projects WHERE id=$1`, id)
	p, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Project{}, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	items, err := s.ListItems(ctx, id)
	if err != nil {
		return p, err
	}
	p.Items = items
	if p.AssetID != nil {
		a, err := s.GetAsset(ctx, *p.AssetID)
		if err == nil {
			p.Asset = &a
		}
	}
	return p, nil
}

func (s *Store) CreateProject(ctx context.Context, in models.ProjectInput) (models.Project, error) {
	if strings.TrimSpace(in.Name) == "" {
		return models.Project{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if in.TargetAmount < 0 {
		return models.Project{}, fmt.Errorf("%w: target must be >= 0", ErrInvalid)
	}
	if in.Color == "" {
		in.Color = "#34d399"
	}
	id := uuid.NewString()
	deadline, err := parseDate(in.Deadline)
	if err != nil {
		return models.Project{}, err
	}
	base := projectBase(in)
	if err := CheckMoney(base); err != nil {
		return models.Project{}, err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO projects (id, name, description, target_amount, current_amount, base_amount, deadline, color, status, result_asset_type)
VALUES ($1,$2,$3,$4,0,$4,$5,$6,'active',$7)`,
		id, strings.TrimSpace(in.Name), in.Description, base, deadline, in.Color, in.ResultAssetType)
	if err != nil {
		return models.Project{}, err
	}
	return s.GetProject(ctx, id)
}

func (s *Store) UpdateProject(ctx context.Context, id string, in models.ProjectInput) (models.Project, error) {
	deadline, err := parseDate(in.Deadline)
	if err != nil {
		return models.Project{}, err
	}
	base := projectBase(in)
	if err := CheckMoney(base); err != nil {
		return models.Project{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Project{}, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `
UPDATE projects SET name=$2, description=$3, base_amount=$4, deadline=$5, color=$6, result_asset_type=$7, updated_at=NOW()
WHERE id=$1`, id, strings.TrimSpace(in.Name), in.Description, base, deadline, in.Color, in.ResultAssetType)
	if err != nil {
		return models.Project{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.Project{}, ErrNotFound
	}
	if err := syncProjectTarget(ctx, tx, id); err != nil {
		return models.Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Project{}, err
	}
	return s.GetProject(ctx, id)
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Contribute(ctx context.Context, projectID string, in models.ContributeInput) (models.Project, error) {
	if in.Amount <= 0 {
		return models.Project{}, fmt.Errorf("%w: amount must be positive", ErrInvalid)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Project{}, err
	}
	defer tx.Rollback()

	var bal int64
	err = tx.QueryRowContext(ctx, `SELECT balance FROM accounts WHERE id=$1 FOR UPDATE`, in.AccountID).Scan(&bal)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Project{}, fmt.Errorf("%w: account", ErrNotFound)
	}
	if err != nil {
		return models.Project{}, err
	}
	if bal < in.Amount {
		return models.Project{}, ErrInsufficient
	}

	var current, target int64
	err = tx.QueryRowContext(ctx, `SELECT current_amount, target_amount FROM projects WHERE id=$1 FOR UPDATE`, projectID).Scan(&current, &target)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Project{}, fmt.Errorf("%w: project", ErrNotFound)
	}
	if err != nil {
		return models.Project{}, err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET balance=balance-$2, updated_at=NOW() WHERE id=$1`, in.AccountID, in.Amount); err != nil {
		return models.Project{}, err
	}
	status := "active"
	if target > 0 && current+in.Amount >= target {
		status = "completed"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE projects SET current_amount=current_amount+$2, status=$3, updated_at=NOW() WHERE id=$1`, projectID, in.Amount, status); err != nil {
		return models.Project{}, err
	}
	desc := in.Description
	if desc == "" {
		desc = "هزینه پروژه"
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, project_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,$3,'expense',$4,'پروژه',$5,NOW(),NOW())`,
		uuid.NewString(), in.AccountID, projectID, in.Amount, desc); err != nil {
		return models.Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Project{}, err
	}
	return s.GetProject(ctx, projectID)
}

func (s *Store) WithdrawProject(ctx context.Context, projectID string, in models.ContributeInput) (models.Project, error) {
	if in.Amount <= 0 {
		return models.Project{}, fmt.Errorf("%w: amount must be positive", ErrInvalid)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Project{}, err
	}
	defer tx.Rollback()

	var current int64
	err = tx.QueryRowContext(ctx, `SELECT current_amount FROM projects WHERE id=$1 FOR UPDATE`, projectID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Project{}, fmt.Errorf("%w: project", ErrNotFound)
	}
	if err != nil {
		return models.Project{}, err
	}
	if current < in.Amount {
		return models.Project{}, ErrInsufficient
	}
	var exists string
	err = tx.QueryRowContext(ctx, `SELECT id FROM accounts WHERE id=$1 FOR UPDATE`, in.AccountID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Project{}, fmt.Errorf("%w: account", ErrNotFound)
	}
	if err != nil {
		return models.Project{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE projects SET current_amount=current_amount-$2, status='active', updated_at=NOW() WHERE id=$1`, projectID, in.Amount); err != nil {
		return models.Project{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET balance=balance+$2, updated_at=NOW() WHERE id=$1`, in.AccountID, in.Amount); err != nil {
		return models.Project{}, err
	}
	desc := in.Description
	if desc == "" {
		desc = "برداشت از پروژه"
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, project_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,$3,'withdrawal',$4,'پروژه',$5,NOW(),NOW())`,
		uuid.NewString(), in.AccountID, projectID, in.Amount, desc); err != nil {
		return models.Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Project{}, err
	}
	return s.GetProject(ctx, projectID)
}

func (s *Store) ListTransactions(ctx context.Context, limit int, txType, accountID, projectID string) ([]models.Transaction, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `
SELECT t.id, t.account_id, t.to_account_id, t.project_id, t.type, t.amount, t.category, t.description, t.occurred_at, t.created_at,
       COALESCE(a.name,''), COALESCE(b.name,''), COALESCE(p.name,'')
FROM transactions t
LEFT JOIN accounts a ON a.id = t.account_id
LEFT JOIN accounts b ON b.id = t.to_account_id
LEFT JOIN projects p ON p.id = t.project_id
WHERE 1=1`
	args := []any{}
	n := 1
	if txType != "" {
		query += fmt.Sprintf(" AND t.type=$%d", n)
		args = append(args, txType)
		n++
	}
	if accountID != "" {
		query += fmt.Sprintf(" AND (t.account_id=$%d OR t.to_account_id=$%d)", n, n)
		args = append(args, accountID)
		n++
	}
	if projectID != "" {
		query += fmt.Sprintf(" AND t.project_id=$%d", n)
		args = append(args, projectID)
		n++
	}
	query += fmt.Sprintf(" ORDER BY t.occurred_at DESC, t.created_at DESC LIMIT $%d", n)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Transaction, 0)
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateTransaction(ctx context.Context, in models.TransactionInput) (models.Transaction, error) {
	if err := CheckMoneyPositive(in.Amount); err != nil {
		return models.Transaction{}, err
	}
	switch in.Type {
	case "income", "expense", "transfer":
	default:
		return models.Transaction{}, fmt.Errorf("%w: invalid type", ErrInvalid)
	}
	occurred := time.Now()
	if in.OccurredAt != nil && *in.OccurredAt != "" {
		parsed, err := time.Parse(time.RFC3339, *in.OccurredAt)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", *in.OccurredAt)
			if err != nil {
				return models.Transaction{}, fmt.Errorf("%w: invalid date", ErrInvalid)
			}
		}
		occurred = parsed
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Transaction{}, err
	}
	defer tx.Rollback()

	id := uuid.NewString()
	switch in.Type {
	case "income":
		if in.AccountID == nil {
			return models.Transaction{}, fmt.Errorf("%w: account required", ErrInvalid)
		}
		if err := changeBalance(ctx, tx, *in.AccountID, in.Amount); err != nil {
			return models.Transaction{}, err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, project_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,$3,'income',$4,$5,$6,$7,$7)`,
			id, *in.AccountID, nullString(in.ProjectID), in.Amount, in.Category, in.Description, occurred); err != nil {
			return models.Transaction{}, err
		}
	case "expense":
		if in.AccountID == nil {
			return models.Transaction{}, fmt.Errorf("%w: account required", ErrInvalid)
		}
		itemID := in.ItemID
		if itemID != nil && *itemID == "" {
			itemID = nil
		}
		if itemID != nil {
			if in.ProjectID == nil || *in.ProjectID == "" {
				return models.Transaction{}, fmt.Errorf("%w: item requires project", ErrInvalid)
			}
			var owner string
			err := tx.QueryRowContext(ctx, `SELECT project_id FROM project_items WHERE id=$1 FOR UPDATE`, *itemID).Scan(&owner)
			if errors.Is(err, sql.ErrNoRows) {
				return models.Transaction{}, fmt.Errorf("%w: project item", ErrNotFound)
			}
			if err != nil {
				return models.Transaction{}, err
			}
			if owner != *in.ProjectID {
				return models.Transaction{}, fmt.Errorf("%w: item does not belong to project", ErrInvalid)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE project_items SET paid_amount=paid_amount+$2, updated_at=NOW() WHERE id=$1`, *itemID, in.Amount); err != nil {
				return models.Transaction{}, err
			}
		}
		if err := changeBalance(ctx, tx, *in.AccountID, -in.Amount); err != nil {
			return models.Transaction{}, err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, project_id, item_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,$3,$4,'expense',$5,$6,$7,$8,$8)`,
			id, *in.AccountID, nullString(in.ProjectID), nullString(itemID), in.Amount, in.Category, in.Description, occurred); err != nil {
			return models.Transaction{}, err
		}
		if in.ProjectID != nil && *in.ProjectID != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE projects SET current_amount=current_amount+$2, updated_at=NOW() WHERE id=$1`, *in.ProjectID, in.Amount); err != nil {
				return models.Transaction{}, err
			}
		}
	case "transfer":
		if in.AccountID == nil || in.ToAccountID == nil {
			return models.Transaction{}, fmt.Errorf("%w: both accounts required", ErrInvalid)
		}
		if *in.AccountID == *in.ToAccountID {
			return models.Transaction{}, fmt.Errorf("%w: accounts must differ", ErrInvalid)
		}
		if err := changeBalance(ctx, tx, *in.AccountID, -in.Amount); err != nil {
			return models.Transaction{}, err
		}
		if err := changeBalance(ctx, tx, *in.ToAccountID, in.Amount); err != nil {
			return models.Transaction{}, err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, to_account_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,$3,'transfer',$4,$5,$6,$7,$7)`,
			id, *in.AccountID, *in.ToAccountID, in.Amount, in.Category, in.Description, occurred); err != nil {
			return models.Transaction{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Transaction{}, err
	}
	return s.GetTransaction(ctx, id)
}

func (s *Store) GetTransaction(ctx context.Context, id string) (models.Transaction, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT t.id, t.account_id, t.to_account_id, t.project_id, t.type, t.amount, t.category, t.description, t.occurred_at, t.created_at,
       COALESCE(a.name,''), COALESCE(b.name,''), COALESCE(p.name,'')
FROM transactions t
LEFT JOIN accounts a ON a.id = t.account_id
LEFT JOIN accounts b ON b.id = t.to_account_id
LEFT JOIN projects p ON p.id = t.project_id
WHERE t.id=$1`, id)
	t, err := scanTransaction(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Transaction{}, ErrNotFound
	}
	return t, err
}

func (s *Store) DeleteTransaction(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		accountID, toAccountID, projectID, itemID sql.NullString
		txType                                    string
		amount                                    int64
	)
	err = tx.QueryRowContext(ctx, `
SELECT account_id, to_account_id, project_id, item_id, type, amount FROM transactions WHERE id=$1 FOR UPDATE`, id).
		Scan(&accountID, &toAccountID, &projectID, &itemID, &txType, &amount)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	switch txType {
	case "income", "opening":
		if accountID.Valid {
			if err := changeBalance(ctx, tx, accountID.String, -amount); err != nil {
				return err
			}
		}
	case "expense":
		if accountID.Valid {
			if err := changeBalanceAllowNegative(ctx, tx, accountID.String, amount); err != nil {
				return err
			}
		}
		if projectID.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE projects SET current_amount=GREATEST(current_amount-$2,0), status='active', updated_at=NOW() WHERE id=$1`, projectID.String, amount); err != nil {
				return err
			}
		}
		if itemID.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE project_items SET paid_amount=GREATEST(paid_amount-$2,0), updated_at=NOW() WHERE id=$1`, itemID.String, amount); err != nil {
				return err
			}
		}
	case "transfer":
		if accountID.Valid {
			if err := changeBalanceAllowNegative(ctx, tx, accountID.String, amount); err != nil {
				return err
			}
		}
		if toAccountID.Valid {
			if err := changeBalance(ctx, tx, toAccountID.String, -amount); err != nil {
				return err
			}
		}
	case "contribution":
		if accountID.Valid {
			if err := changeBalanceAllowNegative(ctx, tx, accountID.String, amount); err != nil {
				return err
			}
		}
		if projectID.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE projects SET current_amount=GREATEST(current_amount-$2,0), status='active', updated_at=NOW() WHERE id=$1`, projectID.String, amount); err != nil {
				return err
			}
		}
	case "withdrawal":
		if accountID.Valid {
			if err := changeBalance(ctx, tx, accountID.String, -amount); err != nil {
				return err
			}
		}
		if projectID.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE projects SET current_amount=current_amount+$2, updated_at=NOW() WHERE id=$1`, projectID.String, amount); err != nil {
				return err
			}
		}
	case "adjustment":
		if accountID.Valid {
			if err := changeBalanceAllowNegative(ctx, tx, accountID.String, -amount); err != nil {
				return err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM transactions WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Dashboard(ctx context.Context) (models.Dashboard, error) {
	var d models.Dashboard
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(balance),0), COUNT(*) FROM accounts`).Scan(&d.Liquid, &d.AccountCount)
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(current_amount),0), COUNT(*) FILTER (WHERE status='active') FROM projects`).Scan(&d.ProjectSpend, &d.ProjectCount)
	_ = s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(
  CASE
    WHEN has_schedule THEN COALESCE((
      SELECT SUM(i.amount) FROM debt_installments i
      WHERE i.debt_id = d.id AND i.status = 'pending'
    ), 0)
    ELSE remaining
  END
), 0)
FROM debts d WHERE status='active'`).Scan(&d.DebtsRemaining)

	_ = s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(amount),0) FROM transactions
WHERE type='income' AND account_id IS NOT NULL AND occurred_at >= date_trunc('month', NOW())`).Scan(&d.MonthlyIncome)
	_ = s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(amount),0) FROM transactions
WHERE type IN ('expense','contribution') AND account_id IS NOT NULL AND occurred_at >= date_trunc('month', NOW())`).Scan(&d.MonthlyExpense)

	accounts, err := s.ListAccounts(ctx)
	if err != nil {
		return d, err
	}
	d.Accounts = accounts
	projects, err := s.ListProjects(ctx)
	if err != nil {
		return d, err
	}
	d.Projects = projects
	assets, err := s.ListAssets(ctx)
	if err != nil {
		return d, err
	}
	d.Assets = assets
	d.AssetsTotal = liveAssetsTotal(ctx, assets)
	d.NetWorth = d.Liquid + d.AssetsTotal - d.DebtsRemaining
	upcoming, err := s.UpcomingInstallments(ctx, 6)
	if err != nil {
		return d, err
	}
	d.Upcoming = upcoming
	recent, err := s.ListTransactions(ctx, 8, "", "", "")
	if err != nil {
		return d, err
	}
	d.Recent = recent

	rows, err := s.db.QueryContext(ctx, `
SELECT to_char(day, 'YYYY-MM-DD'),
       COALESCE(SUM(income),0), COALESCE(SUM(expense),0)
FROM (
  SELECT date_trunc('day', occurred_at) AS day,
         CASE WHEN type='income' THEN amount ELSE 0 END AS income,
         CASE WHEN type IN ('expense','contribution') THEN amount ELSE 0 END AS expense
  FROM transactions
  WHERE occurred_at >= NOW() - INTERVAL '30 days'
    AND account_id IS NOT NULL
    AND type IN ('income','expense','contribution')
) q
GROUP BY day ORDER BY day`)
	if err != nil {
		return d, err
	}
	d.Cashflow = []models.DailyPoint{}
	for rows.Next() {
		var p models.DailyPoint
		if err := rows.Scan(&p.Date, &p.Income, &p.Expense); err != nil {
			rows.Close()
			return d, err
		}
		d.Cashflow = append(d.Cashflow, p)
	}
	rows.Close()

	crows, err := s.db.QueryContext(ctx, `
SELECT COALESCE(NULLIF(category,''),'سایر'), COALESCE(SUM(amount),0)
FROM transactions
WHERE type IN ('expense','contribution') AND account_id IS NOT NULL AND occurred_at >= date_trunc('month', NOW())
GROUP BY 1 ORDER BY 2 DESC LIMIT 8`)
	if err != nil {
		return d, err
	}
	d.CategorySpend = []models.CategoryPoint{}
	for crows.Next() {
		var p models.CategoryPoint
		if err := crows.Scan(&p.Category, &p.Amount); err != nil {
			crows.Close()
			return d, err
		}
		d.CategorySpend = append(d.CategorySpend, p)
	}
	crows.Close()

	mrows, err := s.db.QueryContext(ctx, `
SELECT to_char(date_trunc('month', occurred_at), 'YYYY-MM'),
       COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END),0),
       COALESCE(SUM(CASE WHEN type IN ('expense','contribution') THEN amount ELSE 0 END),0)
FROM transactions
WHERE occurred_at >= date_trunc('month', NOW()) - INTERVAL '5 months'
  AND account_id IS NOT NULL
  AND type IN ('income','expense','contribution')
GROUP BY 1 ORDER BY 1`)
	if err != nil {
		return d, err
	}
	d.Monthly = []models.MonthlyPoint{}
	for mrows.Next() {
		var p models.MonthlyPoint
		if err := mrows.Scan(&p.Month, &p.Income, &p.Expense); err != nil {
			mrows.Close()
			return d, err
		}
		d.Monthly = append(d.Monthly, p)
	}
	mrows.Close()
	return d, nil
}

func liveAssetsTotal(ctx context.Context, assets []models.Asset) int64 {
	var book int64
	for _, a := range assets {
		book += a.Value
	}
	r, err := rates.Get(ctx)
	if err != nil {
		return book
	}
	var live int64
	for _, a := range assets {
		live += rates.AssetValue(a.Type, a.Quantity, a.Value, &r)
	}
	return live
}

func changeBalance(ctx context.Context, tx *sql.Tx, accountID string, delta int64) error {
	var bal int64
	err := tx.QueryRowContext(ctx, `SELECT balance FROM accounts WHERE id=$1 FOR UPDATE`, accountID).Scan(&bal)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: account", ErrNotFound)
	}
	if err != nil {
		return err
	}
	if bal+delta < 0 {
		return ErrInsufficient
	}
	if delta > 0 && bal > models.MaxMoney-delta {
		return fmt.Errorf("%w: amount too large", ErrInvalid)
	}
	_, err = tx.ExecContext(ctx, `UPDATE accounts SET balance=balance+$2, updated_at=NOW() WHERE id=$1`, accountID, delta)
	return err
}

func changeBalanceAllowNegative(ctx context.Context, tx *sql.Tx, accountID string, delta int64) error {
	res, err := tx.ExecContext(ctx, `UPDATE accounts SET balance=balance+$2, updated_at=NOW() WHERE id=$1`, accountID, delta)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: account", ErrNotFound)
	}
	return nil
}

func projectBase(in models.ProjectInput) int64 {
	if in.BaseAmount > 0 {
		return in.BaseAmount
	}
	return in.TargetAmount
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func syncProjectTarget(ctx context.Context, exec execer, projectID string) error {
	_, err := exec.ExecContext(ctx, `
WITH sums AS (
  SELECT COALESCE(SUM(planned_amount), 0) AS items FROM project_items WHERE project_id = $1
)
UPDATE projects p SET
  target_amount = p.base_amount + s.items,
  status = CASE
    WHEN p.asset_id IS NOT NULL THEN p.status
    WHEN p.base_amount + s.items > 0 AND p.current_amount >= p.base_amount + s.items THEN 'completed'
    ELSE 'active'
  END,
  updated_at = NOW()
FROM sums s
WHERE p.id = $1`, projectID)
	return err
}

func parseDate(raw *string) (any, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*raw))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid deadline", ErrInvalid)
	}
	return t, nil
}

func nullString(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAccount(s scanner) (models.Account, error) {
	var a models.Account
	err := s.Scan(&a.ID, &a.Name, &a.BankName, &a.AccountNumber, &a.Type, &a.Balance, &a.Color, &a.Icon, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func scanProject(s scanner) (models.Project, error) {
	var p models.Project
	var deadline pq.NullTime
	var assetID sql.NullString
	err := s.Scan(&p.ID, &p.Name, &p.Description, &p.TargetAmount, &p.CurrentAmount, &p.BaseAmount, &deadline, &p.Color, &p.Status, &p.CreatedAt, &p.UpdatedAt, &assetID, &p.ResultAssetType)
	if deadline.Valid {
		t := deadline.Time
		p.Deadline = &t
	}
	if assetID.Valid {
		p.AssetID = &assetID.String
	}
	return p, err
}

func scanTransaction(s scanner) (models.Transaction, error) {
	var t models.Transaction
	var accountID, toID, projectID sql.NullString
	err := s.Scan(&t.ID, &accountID, &toID, &projectID, &t.Type, &t.Amount, &t.Category, &t.Description, &t.OccurredAt, &t.CreatedAt, &t.AccountName, &t.ToName, &t.ProjectName)
	if accountID.Valid {
		t.AccountID = &accountID.String
	}
	if toID.Valid {
		t.ToAccountID = &toID.String
	}
	if projectID.Valid {
		t.ProjectID = &projectID.String
	}
	return t, err
}
