package notify

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"pvmoney/internal/models"
	"pvmoney/internal/store"
)

func (b *Bot) startTx(ctx context.Context, chat int64, msgID int, kind string) {
	accounts, err := b.store.ListAccounts(ctx)
	if err != nil || len(accounts) == 0 {
		b.show(chat, msgID, b.t("empty.acc"), [][]btn{backRow(b)})
		return
	}
	if kind == "transfer" && len(accounts) < 2 {
		b.show(chat, msgID, b.t("tx.needTwo"), [][]btn{backRow(b)})
		return
	}
	s := b.get(chat)
	*s = session{Kind: kind, Step: "tx_acc"}
	ids, rows := accButtons(accounts, "tx:a:")
	s.Opts = ids
	rows = append(rows, []btn{{Text: b.t("btn.cancel"), Data: "tx:x"}})
	title := b.t("tx.pickAcc")
	if kind == "transfer" {
		title = "↔️ " + title
	} else {
		title = b.t("type."+kind) + "\n" + title
	}
	if msgID == 0 {
		_ = b.sendTo(chat, title, rows, false)
		return
	}
	b.show(chat, msgID, title, rows)
}

func (b *Bot) pickAccount(ctx context.Context, chat int64, msgID int, idx string, dest bool) {
	s := b.get(chat)
	i, err := strconv.Atoi(idx)
	if err != nil || i < 0 || i >= len(s.Opts) {
		return
	}
	id := s.Opts[i]
	if dest {
		if id == s.AccountID {
			return
		}
		s.ToID = id
		b.askAmount(chat, msgID)
		return
	}
	s.AccountID = id
	if s.Kind == "transfer" {
		accounts, _ := b.store.ListAccounts(ctx)
		filtered := make([]models.Account, 0)
		for _, a := range accounts {
			if a.ID != id {
				filtered = append(filtered, a)
			}
		}
		ids, rows := accButtons(filtered, "tx:t:")
		s.Opts = ids
		s.Step = "tx_to"
		rows = append(rows, []btn{{Text: b.t("btn.cancel"), Data: "tx:x"}})
		b.show(chat, msgID, b.t("tx.pickTo"), rows)
		return
	}
	b.askCategory(chat, msgID)
}

func (b *Bot) askCategory(chat int64, msgID int) {
	s := b.get(chat)
	s.Step = "tx_cat"
	list := expenseCats
	if s.Kind == "income" {
		list = incomeCats
	}
	lang := b.lang()
	btns := make([]btn, 0, len(list))
	ids := make([]string, 0, len(list))
	for i, c := range list {
		ids = append(ids, c.ID)
		btns = append(btns, btn{Text: c.label(lang), Data: "tx:c:" + strconv.Itoa(i)})
	}
	s.Opts = ids
	rows := chunkBtns(btns, 2)
	rows = append(rows, []btn{{Text: b.t("btn.skip"), Data: "tx:skipc"}, {Text: b.t("btn.cancel"), Data: "tx:x"}})
	b.show(chat, msgID, b.t("tx.pickCat"), rows)
}

func (b *Bot) pickCategory(_ context.Context, chat int64, msgID int, idx string) {
	s := b.get(chat)
	if idx != "" {
		i, err := strconv.Atoi(idx)
		if err == nil && i >= 0 && i < len(s.Opts) {
			s.Category = s.Opts[i]
		}
	} else {
		s.Category = ""
	}
	b.askAmount(chat, msgID)
}

func (b *Bot) askAmount(chat int64, msgID int) {
	s := b.get(chat)
	s.Step = "tx_amount"
	b.show(chat, msgID, b.t("tx.amount"), [][]btn{{{Text: b.t("btn.cancel"), Data: "tx:x"}}})
}

func (b *Bot) askNote(chat int64) {
	s := b.get(chat)
	s.Step = "tx_note"
	_ = b.sendTo(chat, b.t("tx.note"), [][]btn{
		{{Text: b.t("btn.skip"), Data: "tx:skipn"}, {Text: b.t("btn.cancel"), Data: "tx:x"}},
	}, false)
}

func (b *Bot) askConfirm(ctx context.Context, chat int64, msgID int) {
	s := b.get(chat)
	s.Step = "tx_ok"
	acc, _ := b.store.GetAccount(ctx, s.AccountID)
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s\n\n", b.t("tx.confirm"))
	fmt.Fprintf(&sb, "%s\n%s\n", b.t("type."+s.Kind), toman(s.Amount))
	fmt.Fprintf(&sb, "%s", acc.Name)
	if s.Kind == "transfer" && s.ToID != "" {
		to, _ := b.store.GetAccount(ctx, s.ToID)
		fmt.Fprintf(&sb, " → %s", to.Name)
	}
	if s.Category != "" {
		fmt.Fprintf(&sb, "\n%s", catName(s.Category, b.lang()))
	}
	if s.Note != "" {
		fmt.Fprintf(&sb, "\n%s", s.Note)
	}
	rows := [][]btn{{{Text: b.t("btn.ok"), Data: "tx:ok"}, {Text: b.t("btn.cancel"), Data: "tx:x"}}}
	if msgID > 0 {
		b.show(chat, msgID, sb.String(), rows)
		return
	}
	_ = b.sendTo(chat, sb.String(), rows, false)
}

func catName(id, lang string) string {
	for _, c := range append(expenseCats, incomeCats...) {
		if c.ID == id {
			return c.label(lang)
		}
	}
	return id
}

func (b *Bot) commitTx(ctx context.Context, chat int64, msgID int) {
	s := b.get(chat)
	accID := s.AccountID
	in := models.TransactionInput{
		AccountID:   &accID,
		Type:        s.Kind,
		Amount:      s.Amount,
		Category:    s.Category,
		Description: s.Note,
	}
	if s.Kind == "transfer" {
		to := s.ToID
		in.ToAccountID = &to
		in.Category = "transfer"
	}
	item, err := b.store.CreateTransaction(ctx, in)
	if err != nil {
		msg := err.Error()
		if errors.Is(err, store.ErrInsufficient) {
			msg = "Insufficient balance / موجودی کافی نیست / Nicht genug Guthaben"
		}
		b.show(chat, msgID, msg, [][]btn{backRow(b)})
		return
	}
	b.clear(chat)
	acc, _ := b.store.GetAccount(ctx, accID)
	text := b.t("tx.saved") + "\n\n" + formatTxLine(item) + "\n\n💳 " + acc.Name + ": " + toman(acc.Balance)
	b.show(chat, msgID, text, navRow(b))
}

func formatTxLine(t models.Transaction) string {
	sign := ""
	icon := "•"
	switch t.Type {
	case "income":
		sign, icon = "+", "🟢"
	case "expense":
		sign, icon = "−", "🔴"
	case "transfer":
		icon = "↔️"
	}
	line := fmt.Sprintf("%s %s%s", icon, sign, toman(t.Amount))
	if t.AccountName != "" {
		line += "\n" + t.AccountName
		if t.ToName != "" {
			line += " → " + t.ToName
		}
	}
	if t.Description != "" {
		line += "\n" + t.Description
	}
	return line
}

func (b *Bot) startPay(ctx context.Context, chat int64, msgID int, idx string) {
	s := b.get(chat)
	i, err := strconv.Atoi(idx)
	if err != nil || i < 0 || i >= len(s.Opts) {
		return
	}
	debt, err := b.store.GetDebt(ctx, s.Opts[i])
	if err != nil {
		return
	}
	var inst *models.Installment
	for n := range debt.Installments {
		it := debt.Installments[n]
		if it.Status == "pending" || it.Status == "overdue" {
			inst = &it
			break
		}
	}
	accounts, _ := b.store.ListAccounts(ctx)
	if len(accounts) == 0 {
		b.show(chat, msgID, b.t("empty.acc"), [][]btn{backRow(b)})
		return
	}
	ns := session{Step: "pay_acc", DebtID: debt.ID, Amount: debt.Remaining, Kind: "pay"}
	if inst != nil {
		ns.InstID = inst.ID
		ns.Amount = inst.Amount
	}
	ids, rows := accButtons(accounts, "d:a:")
	ns.Opts = ids
	b.mu.Lock()
	b.sess[chat] = &ns
	b.mu.Unlock()
	if ns.Amount <= 0 {
		b.show(chat, msgID, b.t("pay.none"), [][]btn{backRow(b)})
		return
	}
	title := fmt.Sprintf("%s\n%s · %s", b.t("pay.pickAcc"), debt.Name, toman(ns.Amount))
	rows = append(rows, []btn{{Text: b.t("btn.cancel"), Data: "tx:x"}})
	b.show(chat, msgID, title, rows)
}

func (b *Bot) pickPayAccount(ctx context.Context, chat int64, msgID int, idx string) {
	s := b.get(chat)
	i, err := strconv.Atoi(idx)
	if err != nil || i < 0 || i >= len(s.Opts) {
		return
	}
	s.AccountID = s.Opts[i]
	s.Step = "pay_ok"
	acc, _ := b.store.GetAccount(ctx, s.AccountID)
	debt, _ := b.store.GetDebt(ctx, s.DebtID)
	text := fmt.Sprintf("%s\n\n%s\n%s\n💳 %s", b.t("pay.confirm"), debt.Name, toman(s.Amount), acc.Name)
	b.show(chat, msgID, text, [][]btn{
		{{Text: b.t("btn.ok"), Data: "d:ok"}, {Text: b.t("btn.cancel"), Data: "tx:x"}},
	})
}

func (b *Bot) commitPay(ctx context.Context, chat int64, msgID int) {
	s := b.get(chat)
	in := models.PayInput{AccountID: s.AccountID, Amount: s.Amount, Description: "Telegram"}
	var err error
	if s.InstID != "" {
		_, err = b.store.PayInstallment(ctx, s.DebtID, s.InstID, in)
	} else {
		_, err = b.store.PayDebt(ctx, s.DebtID, in)
	}
	if err != nil {
		b.show(chat, msgID, err.Error(), [][]btn{backRow(b)})
		return
	}
	b.clear(chat)
	b.show(chat, msgID, b.t("pay.ok"), navRow(b))
}
