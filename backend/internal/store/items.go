package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"pvmoney/internal/models"

	"github.com/google/uuid"
)

func (s *Store) ListItems(ctx context.Context, projectID string) ([]models.ProjectItem, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_id, name, planned_amount, paid_amount, notes, created_at, updated_at
FROM project_items WHERE project_id=$1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ProjectItem, 0)
	for rows.Next() {
		var it models.ProjectItem
		if err := rows.Scan(&it.ID, &it.ProjectID, &it.Name, &it.PlannedAmount, &it.PaidAmount, &it.Notes, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) CreateItem(ctx context.Context, projectID string, in models.ItemInput) (models.ProjectItem, error) {
	if strings.TrimSpace(in.Name) == "" {
		return models.ProjectItem{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if err := CheckMoney(in.PlannedAmount); err != nil {
		return models.ProjectItem{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ProjectItem{}, err
	}
	defer tx.Rollback()

	var exists string
	err = tx.QueryRowContext(ctx, `SELECT id FROM projects WHERE id=$1 FOR UPDATE`, projectID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ProjectItem{}, ErrNotFound
	}
	if err != nil {
		return models.ProjectItem{}, err
	}
	id := uuid.NewString()
	_, err = tx.ExecContext(ctx, `
INSERT INTO project_items (id, project_id, name, planned_amount, paid_amount, notes)
VALUES ($1,$2,$3,$4,0,$5)`, id, projectID, strings.TrimSpace(in.Name), in.PlannedAmount, in.Notes)
	if err != nil {
		return models.ProjectItem{}, err
	}
	if err := syncProjectTarget(ctx, tx, projectID); err != nil {
		return models.ProjectItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.ProjectItem{}, err
	}
	return s.getItem(ctx, id)
}

func (s *Store) UpdateItem(ctx context.Context, projectID, itemID string, in models.ItemInput) (models.ProjectItem, error) {
	if err := CheckMoney(in.PlannedAmount); err != nil {
		return models.ProjectItem{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ProjectItem{}, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `
UPDATE project_items SET name=$3, planned_amount=$4, notes=$5, updated_at=NOW()
WHERE id=$1 AND project_id=$2`, itemID, projectID, strings.TrimSpace(in.Name), in.PlannedAmount, in.Notes)
	if err != nil {
		return models.ProjectItem{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.ProjectItem{}, ErrNotFound
	}
	if err := syncProjectTarget(ctx, tx, projectID); err != nil {
		return models.ProjectItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.ProjectItem{}, err
	}
	return s.getItem(ctx, itemID)
}

func (s *Store) DeleteItem(ctx context.Context, projectID, itemID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var paid int64
	err = tx.QueryRowContext(ctx, `SELECT paid_amount FROM project_items WHERE id=$1 AND project_id=$2 FOR UPDATE`, itemID, projectID).Scan(&paid)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if paid > 0 {
		return fmt.Errorf("%w: این هزینه پرداخت شده؛ اول تراکنش را حذف کن", ErrInvalid)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_items WHERE id=$1 AND project_id=$2`, itemID, projectID); err != nil {
		return err
	}
	if err := syncProjectTarget(ctx, tx, projectID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) PayItem(ctx context.Context, projectID, itemID string, in models.PayInput) (models.Project, error) {
	if err := CheckMoneyPositive(in.Amount); err != nil {
		return models.Project{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Project{}, err
	}
	defer tx.Rollback()

	var itemName string
	err = tx.QueryRowContext(ctx, `SELECT name FROM project_items WHERE id=$1 AND project_id=$2 FOR UPDATE`, itemID, projectID).Scan(&itemName)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Project{}, ErrNotFound
	}
	if err != nil {
		return models.Project{}, err
	}

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
	if err != nil {
		return models.Project{}, err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET balance=balance-$2, updated_at=NOW() WHERE id=$1`, in.AccountID, in.Amount); err != nil {
		return models.Project{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE project_items SET paid_amount=paid_amount+$2, updated_at=NOW() WHERE id=$1`, itemID, in.Amount); err != nil {
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
		desc = itemName
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, project_id, item_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,$3,$4,'expense',$5,'پروژه',$6,NOW(),NOW())`,
		uuid.NewString(), in.AccountID, projectID, itemID, in.Amount, desc); err != nil {
		return models.Project{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Project{}, err
	}
	return s.GetProject(ctx, projectID)
}

func (s *Store) getItem(ctx context.Context, id string) (models.ProjectItem, error) {
	var it models.ProjectItem
	err := s.db.QueryRowContext(ctx, `
SELECT id, project_id, name, planned_amount, paid_amount, notes, created_at, updated_at
FROM project_items WHERE id=$1`, id).Scan(&it.ID, &it.ProjectID, &it.Name, &it.PlannedAmount, &it.PaidAmount, &it.Notes, &it.CreatedAt, &it.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return it, ErrNotFound
	}
	return it, err
}
