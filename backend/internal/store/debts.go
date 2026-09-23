package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pvmoney/internal/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func tehran() *time.Location {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		return time.FixedZone("IRST", 3.5*3600)
	}
	return loc
}

func (s *Store) ListDebts(ctx context.Context) ([]models.Debt, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT d.id, d.name, d.type, d.creditor, d.total_amount, d.remaining, d.commission_amount, d.notes, d.color,
       d.has_schedule, d.start_date, d.end_date, d.monthly_amount, d.due_day, d.status, d.created_at, d.updated_at,
       (SELECT MIN(due_date) FROM debt_installments i WHERE i.debt_id=d.id AND i.status='pending')
FROM debts d ORDER BY d.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Debt, 0)
	for rows.Next() {
		d, err := scanDebt(rows, true)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) GetDebt(ctx context.Context, id string) (models.Debt, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT d.id, d.name, d.type, d.creditor, d.total_amount, d.remaining, d.commission_amount, d.notes, d.color,
       d.has_schedule, d.start_date, d.end_date, d.monthly_amount, d.due_day, d.status, d.created_at, d.updated_at,
       (SELECT MIN(due_date) FROM debt_installments i WHERE i.debt_id=d.id AND i.status='pending')
FROM debts d WHERE d.id=$1`, id)
	d, err := scanDebt(row, true)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Debt{}, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	items, err := s.listInstallments(ctx, id)
	if err != nil {
		return d, err
	}
	d.Installments = items
	return d, nil
}

func (s *Store) CreateDebt(ctx context.Context, in models.DebtInput) (models.Debt, error) {
	if strings.TrimSpace(in.Name) == "" {
		return models.Debt{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if err := CheckMoney(in.TotalAmount); err != nil {
		return models.Debt{}, err
	}
	if err := CheckMoney(in.MonthlyAmount); err != nil {
		return models.Debt{}, err
	}
	if err := CheckMoney(in.CommissionAmount); err != nil {
		return models.Debt{}, err
	}
	if in.Type == "" {
		in.Type = "loan"
	}
	if in.Color == "" {
		in.Color = "#fb7185"
	}
	start, err := parseDate(in.StartDate)
	if err != nil {
		return models.Debt{}, err
	}
	end, err := parseDate(in.EndDate)
	if err != nil {
		return models.Debt{}, err
	}
	id := uuid.NewString()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Debt{}, err
	}
	defer tx.Rollback()

	if in.TotalAmount <= 0 {
		return models.Debt{}, fmt.Errorf("%w: total amount required", ErrInvalid)
	}

	remaining := in.TotalAmount
	var amounts []int64
	var dates []time.Time
	if in.HasSchedule {
		st, ok := start.(time.Time)
		if !ok {
			return models.Debt{}, fmt.Errorf("%w: first due date required", ErrInvalid)
		}
		if in.MonthlyAmount <= 0 {
			in.MonthlyAmount = in.TotalAmount
		}
		dueDay := in.DueDay
		if dueDay < 1 || dueDay > 31 {
			dueDay = st.Day()
		}
		in.DueDay = dueDay
		var errPlan error
		amounts, dates, errPlan = planInstallments(in.TotalAmount, in.MonthlyAmount, st, dueDay, in.Count)
		if errPlan != nil {
			return models.Debt{}, errPlan
		}
		remaining = sumAmounts(amounts)
		end = dates[len(dates)-1]
	} else if start != nil {
		if st, ok := start.(time.Time); ok {
			dates = []time.Time{st}
			amounts = []int64{in.TotalAmount}
		}
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO debts (id, name, type, creditor, total_amount, remaining, commission_amount, notes, color, has_schedule, start_date, end_date, monthly_amount, due_day, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'active')`,
		id, strings.TrimSpace(in.Name), in.Type, in.Creditor, in.TotalAmount, remaining, in.CommissionAmount, in.Notes, in.Color, in.HasSchedule, start, end, in.MonthlyAmount, in.DueDay)
	if err != nil {
		return models.Debt{}, err
	}
	prepaid := in.AlreadyPaidCount
	if prepaid < 0 {
		prepaid = 0
	}
	if prepaid > len(amounts) {
		prepaid = len(amounts)
	}
	remaining = 0
	for i, dt := range dates {
		status := "pending"
		var paidAt any
		if i < prepaid {
			status = "paid"
			paidAt = dt
		} else {
			remaining += amounts[i]
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO debt_installments (id, debt_id, amount, due_date, status, paid_at) VALUES ($1,$2,$3,$4,$5,$6)`,
			uuid.NewString(), id, amounts[i], dt, status, paidAt); err != nil {
			return models.Debt{}, err
		}
	}
	status := "active"
	if remaining <= 0 {
		status = "paid"
		remaining = 0
	}
	if _, err := tx.ExecContext(ctx, `UPDATE debts SET remaining=$2, status=$3 WHERE id=$1`, id, remaining, status); err != nil {
		return models.Debt{}, err
	}
	if acc := strings.TrimSpace(deref(in.CommissionAccountID)); acc != "" && in.CommissionAmount > 0 {
		if err := changeBalance(ctx, tx, acc, -in.CommissionAmount); err != nil {
			return models.Debt{}, err
		}
		desc := "کارمزد " + strings.TrimSpace(in.Name)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,'expense',$3,'debt',$4,NOW(),NOW())`, uuid.NewString(), acc, in.CommissionAmount, desc); err != nil {
			return models.Debt{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Debt{}, err
	}
	return s.GetDebt(ctx, id)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *Store) UpdateDebt(ctx context.Context, id string, in models.DebtInput) (models.Debt, error) {
	if strings.TrimSpace(in.Name) == "" {
		return models.Debt{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if err := CheckMoney(in.TotalAmount); err != nil {
		return models.Debt{}, err
	}
	if err := CheckMoney(in.MonthlyAmount); err != nil {
		return models.Debt{}, err
	}
	if err := CheckMoney(in.CommissionAmount); err != nil {
		return models.Debt{}, err
	}
	if in.TotalAmount <= 0 {
		return models.Debt{}, fmt.Errorf("%w: total amount required", ErrInvalid)
	}
	start, err := parseDate(in.StartDate)
	if err != nil {
		return models.Debt{}, err
	}
	end, err := parseDate(in.EndDate)
	if err != nil {
		return models.Debt{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Debt{}, err
	}
	defer tx.Rollback()

	var exists string
	err = tx.QueryRowContext(ctx, `SELECT id FROM debts WHERE id=$1 FOR UPDATE`, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Debt{}, ErrNotFound
	}
	if err != nil {
		return models.Debt{}, err
	}

	var accountPaid int64
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(SUM(amount),0) FROM debt_installments
WHERE debt_id=$1 AND status='paid' AND account_id IS NOT NULL`, id).Scan(&accountPaid); err != nil {
		return models.Debt{}, err
	}
	if in.TotalAmount < accountPaid {
		return models.Debt{}, fmt.Errorf("%w: total is less than already paid", ErrInvalid)
	}

	if _, err := tx.ExecContext(ctx, `
DELETE FROM debt_installments
WHERE debt_id=$1 AND NOT (status='paid' AND account_id IS NOT NULL)`, id); err != nil {
		return models.Debt{}, err
	}

	left := in.TotalAmount - accountPaid
	remaining := left
	if in.Type == "" {
		in.Type = "loan"
	}

	if in.HasSchedule && left > 0 {
		st, ok := start.(time.Time)
		if !ok {
			return models.Debt{}, fmt.Errorf("%w: first due date required", ErrInvalid)
		}
		if in.MonthlyAmount <= 0 {
			in.MonthlyAmount = left
		}
		dueDay := in.DueDay
		if dueDay < 1 || dueDay > 31 {
			dueDay = st.Day()
		}
		in.DueDay = dueDay
		amounts, dates, errPlan := planInstallments(left, in.MonthlyAmount, st, dueDay, in.Count)
		if errPlan != nil {
			return models.Debt{}, errPlan
		}
		end = dates[len(dates)-1]
		prepaid := in.AlreadyPaidCount
		if prepaid < 0 {
			prepaid = 0
		}
		if prepaid > len(amounts) {
			prepaid = len(amounts)
		}
		remaining = 0
		for i, dt := range dates {
			status := "pending"
			var paidAt any
			if i < prepaid {
				status = "paid"
				paidAt = dt
			} else {
				remaining += amounts[i]
			}
			if _, err := tx.ExecContext(ctx, `
INSERT INTO debt_installments (id, debt_id, amount, due_date, status, paid_at) VALUES ($1,$2,$3,$4,$5,$6)`,
				uuid.NewString(), id, amounts[i], dt, status, paidAt); err != nil {
				return models.Debt{}, err
			}
		}
	} else if !in.HasSchedule {
		end = start
		if in.AlreadyPaidCount > 0 {
			remaining = 0
		}
	}

	status := "active"
	if remaining <= 0 {
		status = "paid"
		remaining = 0
	}

	res, err := tx.ExecContext(ctx, `
UPDATE debts SET name=$2, type=$3, creditor=$4, notes=$5, color=$6, commission_amount=$7,
  total_amount=$8, remaining=$9, has_schedule=$10, start_date=$11, end_date=$12,
  monthly_amount=$13, due_day=$14, status=$15, updated_at=NOW()
WHERE id=$1`,
		id, strings.TrimSpace(in.Name), in.Type, in.Creditor, in.Notes, in.Color, in.CommissionAmount,
		in.TotalAmount, remaining, in.HasSchedule, start, end, in.MonthlyAmount, in.DueDay, status)
	if err != nil {
		return models.Debt{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.Debt{}, ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return models.Debt{}, err
	}
	return s.GetDebt(ctx, id)
}

func (s *Store) DeleteDebt(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM debts WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) PayInstallment(ctx context.Context, debtID, instID string, in models.PayInput) (models.Debt, error) {
	if in.Amount <= 0 {
		return models.Debt{}, fmt.Errorf("%w: amount must be positive", ErrInvalid)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Debt{}, err
	}
	defer tx.Rollback()

	var status string
	var due time.Time
	err = tx.QueryRowContext(ctx, `SELECT status, due_date FROM debt_installments WHERE id=$1 AND debt_id=$2 FOR UPDATE`, instID, debtID).Scan(&status, &due)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Debt{}, ErrNotFound
	}
	if err != nil {
		return models.Debt{}, err
	}
	if status == "paid" {
		return models.Debt{}, fmt.Errorf("%w: already paid", ErrInvalid)
	}
	if in.AlreadyPaid {
		if _, err := tx.ExecContext(ctx, `
UPDATE debt_installments SET status='paid', paid_at=NOW(), account_id=NULL WHERE id=$1`, instID); err != nil {
			return models.Debt{}, err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE debts SET remaining=GREATEST(remaining-$2,0),
status = CASE WHEN remaining-$2 <= 0 THEN 'paid' ELSE 'active' END,
updated_at=NOW() WHERE id=$1`, debtID, in.Amount); err != nil {
			return models.Debt{}, err
		}
		if err := tx.Commit(); err != nil {
			return models.Debt{}, err
		}
		return s.GetDebt(ctx, debtID)
	}
	if err := changeBalance(ctx, tx, in.AccountID, -in.Amount); err != nil {
		return models.Debt{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE debt_installments SET status='paid', paid_at=NOW(), account_id=$2 WHERE id=$1`, instID, in.AccountID); err != nil {
		return models.Debt{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE debts SET remaining=GREATEST(remaining-$2,0),
status = CASE WHEN remaining-$2 <= 0 THEN 'paid' ELSE 'active' END,
updated_at=NOW() WHERE id=$1`, debtID, in.Amount); err != nil {
		return models.Debt{}, err
	}
	desc := in.Description
	if desc == "" {
		desc = "Debt installment"
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,'expense',$3,'debt',$4,NOW(),NOW())`, uuid.NewString(), in.AccountID, in.Amount, desc); err != nil {
		return models.Debt{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Debt{}, err
	}
	return s.GetDebt(ctx, debtID)
}

func (s *Store) UnrecordPriorInstallment(ctx context.Context, debtID, instID string) (models.Debt, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Debt{}, err
	}
	defer tx.Rollback()
	var status string
	var accountID sql.NullString
	var amount int64
	err = tx.QueryRowContext(ctx, `
SELECT status, account_id, amount FROM debt_installments WHERE id=$1 AND debt_id=$2 FOR UPDATE`, instID, debtID).
		Scan(&status, &accountID, &amount)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Debt{}, ErrNotFound
	}
	if err != nil {
		return models.Debt{}, err
	}
	if status != "paid" || accountID.Valid {
		return models.Debt{}, fmt.Errorf("%w: only prior payments can be undone", ErrInvalid)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE debt_installments SET status='pending', paid_at=NULL, account_id=NULL WHERE id=$1`, instID); err != nil {
		return models.Debt{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE debts SET remaining=remaining+$2, status='active', updated_at=NOW() WHERE id=$1`, debtID, amount); err != nil {
		return models.Debt{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Debt{}, err
	}
	return s.GetDebt(ctx, debtID)
}

func (s *Store) PayDebt(ctx context.Context, debtID string, in models.PayInput) (models.Debt, error) {
	if in.Amount <= 0 {
		return models.Debt{}, fmt.Errorf("%w: amount must be positive", ErrInvalid)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Debt{}, err
	}
	defer tx.Rollback()

	var remaining int64
	var status string
	err = tx.QueryRowContext(ctx, `SELECT remaining, status FROM debts WHERE id=$1 FOR UPDATE`, debtID).Scan(&remaining, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Debt{}, ErrNotFound
	}
	if err != nil {
		return models.Debt{}, err
	}
	if status == "paid" {
		return models.Debt{}, fmt.Errorf("%w: already paid", ErrInvalid)
	}
	if in.AlreadyPaid {
		if _, err := tx.ExecContext(ctx, `
UPDATE debts SET remaining=GREATEST(remaining-$2,0),
status = CASE WHEN remaining-$2 <= 0 THEN 'paid' ELSE 'active' END,
updated_at=NOW() WHERE id=$1`, debtID, in.Amount); err != nil {
			return models.Debt{}, err
		}
		if err := tx.Commit(); err != nil {
			return models.Debt{}, err
		}
		return s.GetDebt(ctx, debtID)
	}
	if err := changeBalance(ctx, tx, in.AccountID, -in.Amount); err != nil {
		return models.Debt{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE debts SET remaining=GREATEST(remaining-$2,0),
status = CASE WHEN remaining-$2 <= 0 THEN 'paid' ELSE 'active' END,
updated_at=NOW() WHERE id=$1`, debtID, in.Amount); err != nil {
		return models.Debt{}, err
	}
	desc := in.Description
	if desc == "" {
		desc = "Debt payment"
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO transactions (id, account_id, type, amount, category, description, occurred_at, created_at)
VALUES ($1,$2,'expense',$3,'debt',$4,NOW(),NOW())`, uuid.NewString(), in.AccountID, in.Amount, desc); err != nil {
		return models.Debt{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Debt{}, err
	}
	return s.GetDebt(ctx, debtID)
}

func (s *Store) UpcomingInstallments(ctx context.Context, limit int) ([]models.Installment, error) {
	if limit <= 0 {
		limit = 8
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT i.id, i.debt_id, d.name, d.creditor, i.amount, i.due_date, i.paid_at, i.status, i.account_id
FROM debt_installments i
JOIN debts d ON d.id = i.debt_id
WHERE i.status='pending'
ORDER BY i.due_date ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Installment, 0)
	today := time.Now().In(tehran()).Format("2006-01-02")
	for rows.Next() {
		it, err := scanInstallment(rows)
		if err != nil {
			return nil, err
		}
		if it.Status == "pending" && it.DueDate.Format("2006-01-02") < today {
			it.Status = "overdue"
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) PendingReminders(ctx context.Context) ([]models.Installment, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT i.id, i.debt_id, d.name, d.creditor, i.amount, i.due_date, i.paid_at, i.status, i.account_id
FROM debt_installments i
JOIN debts d ON d.id = i.debt_id
WHERE i.status='pending' AND i.due_date <= CURRENT_DATE + INTERVAL '21 days'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Installment, 0)
	for rows.Next() {
		it, err := scanInstallment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) ReminderSent(ctx context.Context, instID, kind string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM debt_reminders WHERE installment_id=$1 AND kind=$2`, instID, kind).Scan(&n)
	return n > 0, err
}

func (s *Store) MarkReminder(ctx context.Context, instID, kind string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO debt_reminders (installment_id, kind) VALUES ($1,$2) ON CONFLICT DO NOTHING`, instID, kind)
	return err
}

func (s *Store) listInstallments(ctx context.Context, debtID string) ([]models.Installment, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT i.id, i.debt_id, d.name, d.creditor, i.amount, i.due_date, i.paid_at, i.status, i.account_id
FROM debt_installments i JOIN debts d ON d.id=i.debt_id
WHERE i.debt_id=$1 ORDER BY i.due_date`, debtID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	today := time.Now().In(tehran()).Format("2006-01-02")
	out := make([]models.Installment, 0)
	for rows.Next() {
		it, err := scanInstallment(rows)
		if err != nil {
			return nil, err
		}
		if it.Status == "pending" && it.DueDate.Format("2006-01-02") < today {
			it.Status = "overdue"
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func generateDueDates(start, end time.Time, dueDay int) []time.Time {
	loc := tehran()
	start = start.In(loc)
	end = end.In(loc)
	y, m, _ := start.Date()
	cur := clampDate(y, m, dueDay, loc)
	if cur.Before(time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)) {
		cur = nextMonth(cur, dueDay)
	}
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)
	out := make([]time.Time, 0)
	for !cur.After(endDay) {
		out = append(out, cur)
		cur = nextMonth(cur, dueDay)
		if len(out) > 360 {
			break
		}
	}
	return out
}

func planInstallments(total, monthly int64, first time.Time, dueDay, count int) ([]int64, []time.Time, error) {
	amounts, err := splitAmounts(total, monthly, count)
	if err != nil {
		return nil, nil, err
	}
	dates := datesFromFirst(first, dueDay, len(amounts))
	if len(dates) != len(amounts) {
		return nil, nil, fmt.Errorf("%w: could not build dates", ErrInvalid)
	}
	return amounts, dates, nil
}

func splitAmounts(total, monthly int64, count int) ([]int64, error) {
	if total <= 0 {
		return nil, fmt.Errorf("%w: total amount required", ErrInvalid)
	}
	if monthly <= 0 || monthly > total {
		monthly = total
	}
	auto := int(total / monthly)
	if total%monthly != 0 {
		auto++
	}
	if auto < 1 {
		auto = 1
	}
	if count <= 0 {
		count = auto
	}
	if count > 360 {
		return nil, fmt.Errorf("%w: too many installments", ErrInvalid)
	}
	if int64(count-1)*monthly >= total {
		count = auto
	}
	last := total - monthly*int64(count-1)
	if last <= 0 {
		return nil, fmt.Errorf("%w: last installment invalid", ErrInvalid)
	}
	out := make([]int64, count)
	for i := 0; i < count-1; i++ {
		out[i] = monthly
	}
	out[count-1] = last
	return out, nil
}

func datesFromFirst(first time.Time, dueDay, count int) []time.Time {
	loc := tehran()
	first = first.In(loc)
	if dueDay < 1 || dueDay > 31 {
		dueDay = first.Day()
	}
	cur := clampDate(first.Year(), first.Month(), dueDay, loc)
	if cur.Before(time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc)) {
		cur = nextMonth(cur, dueDay)
	}
	out := make([]time.Time, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, cur)
		cur = nextMonth(cur, dueDay)
	}
	return out
}

func sumAmounts(items []int64) int64 {
	var n int64
	for _, v := range items {
		n += v
	}
	return n
}

func nextMonth(t time.Time, dueDay int) time.Time {
	y, m, _ := t.Date()
	m++
	if m > 12 {
		m = 1
		y++
	}
	return clampDate(y, m, dueDay, t.Location())
}

func clampDate(y int, m time.Month, day int, loc *time.Location) time.Time {
	if day < 1 {
		day = 1
	}
	last := time.Date(y, m+1, 0, 0, 0, 0, 0, loc).Day()
	if day > last {
		day = last
	}
	return time.Date(y, m, day, 0, 0, 0, 0, loc)
}

func scanDebt(s scanner, withNext bool) (models.Debt, error) {
	var d models.Debt
	var start, end, next pq.NullTime
	var err error
	if withNext {
		err = s.Scan(&d.ID, &d.Name, &d.Type, &d.Creditor, &d.TotalAmount, &d.Remaining, &d.CommissionAmount, &d.Notes, &d.Color,
			&d.HasSchedule, &start, &end, &d.MonthlyAmount, &d.DueDay, &d.Status, &d.CreatedAt, &d.UpdatedAt, &next)
	} else {
		err = s.Scan(&d.ID, &d.Name, &d.Type, &d.Creditor, &d.TotalAmount, &d.Remaining, &d.CommissionAmount, &d.Notes, &d.Color,
			&d.HasSchedule, &start, &end, &d.MonthlyAmount, &d.DueDay, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	}
	if start.Valid {
		t := start.Time
		d.StartDate = &t
	}
	if end.Valid {
		t := end.Time
		d.EndDate = &t
	}
	if next.Valid {
		t := next.Time
		d.NextDue = &t
	}
	d.TotalCost = d.TotalAmount + d.CommissionAmount
	d.NetReceived = d.TotalAmount - d.CommissionAmount
	if d.NetReceived < 0 {
		d.NetReceived = 0
	}
	return d, err
}

func scanInstallment(s scanner) (models.Installment, error) {
	var it models.Installment
	var paid pq.NullTime
	var acc sql.NullString
	err := s.Scan(&it.ID, &it.DebtID, &it.DebtName, &it.Creditor, &it.Amount, &it.DueDate, &paid, &it.Status, &acc)
	if paid.Valid {
		t := paid.Time
		it.PaidAt = &t
	}
	if acc.Valid {
		it.AccountID = &acc.String
	}
	return it, err
}
