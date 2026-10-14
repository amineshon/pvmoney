package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"pvmoney/internal/models"

	"github.com/google/uuid"
)

type repairInst struct {
	id        string
	amount    int64
	due       time.Time
	status    string
	accountID sql.NullString
	paidAt    sql.NullTime
}

func (s *Store) RepairJalaliSchedules(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES ('schedule_jalali_v1', '1') ON CONFLICT (key) DO NOTHING`)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil
	}

	rows, err := tx.QueryContext(ctx, `
SELECT id, name, total_amount, monthly_amount, start_date
FROM debts WHERE has_schedule`)
	if err != nil {
		return err
	}
	type debtRow struct {
		id, name string
		total    int64
		monthly  int64
		start    sql.NullTime
	}
	debts := make([]debtRow, 0)
	for rows.Next() {
		var d debtRow
		if err := rows.Scan(&d.id, &d.name, &d.total, &d.monthly, &d.start); err != nil {
			rows.Close()
			return err
		}
		debts = append(debts, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, d := range debts {
		if err := repairOneSchedule(ctx, tx, d.id, d.name, d.total, d.monthly, d.start); err != nil {
			return fmt.Errorf("%s: %w", d.name, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
CREATE UNIQUE INDEX IF NOT EXISTS uq_debt_installments_due
ON debt_installments (debt_id, due_date)`); err != nil {
		return err
	}

	return tx.Commit()
}

func repairOneSchedule(ctx context.Context, tx *sql.Tx, debtID, name string, total, monthly int64, start sql.NullTime) error {
	if !start.Valid || monthly <= 0 || total <= 0 {
		return nil
	}
	first := start.Time
	dueDay := jalaliDayOf(first)

	items, err := loadRepairInst(ctx, tx, debtID)
	if err != nil {
		return err
	}

	groups := map[string][]repairInst{}
	order := make([]string, 0)
	for _, it := range items {
		key := jalaliMonthKey(it.due)
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], it)
	}

	var paidKept []repairInst
	for _, key := range order {
		keep, extras := pickMonthWinner(groups[key])
		for _, extra := range extras {
			if extra.accountID.Valid && extra.status == "paid" {
				if err := refundDuplicatePay(ctx, tx, extra.accountID.String, extra.amount, name); err != nil {
					return err
				}
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM debt_installments WHERE id=$1`, extra.id); err != nil {
				return err
			}
		}
		if keep.status != "paid" {
			if _, err := tx.ExecContext(ctx, `DELETE FROM debt_installments WHERE id=$1`, keep.id); err != nil {
				return err
			}
			continue
		}
		jy, jm := jalaliYearMonth(keep.due)
		canon := jalaliDate(jy, jm, dueDay)
		if _, err := tx.ExecContext(ctx, `UPDATE debt_installments SET due_date=$2 WHERE id=$1`, keep.id, canon); err != nil {
			return err
		}
		keep.due = canon
		paidKept = append(paidKept, keep)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM debt_installments WHERE debt_id=$1 AND status<>'paid'`, debtID); err != nil {
		return err
	}

	sort.Slice(paidKept, func(i, j int) bool {
		return models.DateOnly(paidKept[i].due) < models.DateOnly(paidKept[j].due)
	})
	var paidSum int64
	for _, it := range paidKept {
		paidSum += it.amount
	}
	left := total - paidSum
	if left < 0 {
		left = 0
	}

	next := jalaliDateFrom(first, dueDay)
	if len(paidKept) > 0 {
		next = addJalaliMonths(paidKept[len(paidKept)-1].due, 1)
	}
	if left > 0 {
		amounts, dates, errPlan := planInstallments(left, monthly, next, dueDay, 0)
		if errPlan != nil {
			return errPlan
		}
		if _, err := insertScheduleRows(ctx, tx, debtID, amounts, dates, 0); err != nil {
			return err
		}
		if len(dates) > 0 {
			if _, err := tx.ExecContext(ctx, `UPDATE debts SET end_date=$2 WHERE id=$1`, debtID, dates[len(dates)-1]); err != nil {
				return err
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE debts SET due_day=$2, start_date=$3, updated_at=NOW() WHERE id=$1`,
		debtID, dueDay, first); err != nil {
		return err
	}
	return refreshDebtRemaining(ctx, tx, debtID)
}

func jalaliDateFrom(t time.Time, dueDay int) time.Time {
	y, m, d := dateYMD(t)
	jy, jm, _ := gregorianToJalali(y, m, d)
	return jalaliDate(jy, jm, dueDay)
}

func loadRepairInst(ctx context.Context, tx *sql.Tx, debtID string) ([]repairInst, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, amount, due_date, status, account_id, paid_at
FROM debt_installments WHERE debt_id=$1
ORDER BY due_date, paid_at NULLS LAST, id`, debtID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]repairInst, 0)
	for rows.Next() {
		var it repairInst
		if err := rows.Scan(&it.id, &it.amount, &it.due, &it.status, &it.accountID, &it.paidAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func pickMonthWinner(items []repairInst) (repairInst, []repairInst) {
	if len(items) == 1 {
		return items[0], nil
	}
	rank := func(it repairInst) int {
		if it.status == "paid" && it.accountID.Valid {
			return 2
		}
		if it.status == "paid" {
			return 1
		}
		return 0
	}
	keep := items[0]
	for _, it := range items[1:] {
		if rank(it) > rank(keep) {
			keep = it
			continue
		}
		if rank(it) == rank(keep) && it.paidAt.Valid && (!keep.paidAt.Valid || it.paidAt.Time.Before(keep.paidAt.Time)) {
			keep = it
		}
	}
	extras := make([]repairInst, 0, len(items)-1)
	for _, it := range items {
		if it.id == keep.id {
			continue
		}
		extras = append(extras, it)
	}
	return keep, extras
}

func refundDuplicatePay(ctx context.Context, tx *sql.Tx, accountID string, amount int64, debtName string) error {
	if accountID == "" || amount <= 0 {
		return nil
	}
	if err := changeBalance(ctx, tx, accountID, amount); err != nil {
		return err
	}
	desc := "برگشت پرداخت تکراری " + debtName
	_, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,'income',$3,'debt',$4,NOW(),NOW())`, uuid.NewString(), accountID, amount, desc)
	return err
}
