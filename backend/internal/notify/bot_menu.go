package notify

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"pvmoney/internal/models"
)

func (b *Bot) onMessage(ctx context.Context, chat int64, text string) {
	s := b.get(chat)
	low := strings.ToLower(strings.TrimSpace(text))

	if strings.HasPrefix(low, "/start") || strings.HasPrefix(low, "/menu") || b.isBtn(text, "kb.menu") {
		b.clear(chat)
		_ = b.sendHome(ctx, chat, 0, true)
		return
	}
	if strings.HasPrefix(low, "/cancel") || b.isBtn(text, "btn.cancel") {
		b.clear(chat)
		_ = b.sendTo(chat, b.t("tx.cancelled"), nil, true)
		_ = b.sendHome(ctx, chat, 0, false)
		return
	}

	if s.Step == "tx_amount" {
		n, err := parseAmount(text)
		if err != nil || n <= 0 {
			_ = b.sendTo(chat, b.t("tx.badAmt"), [][]btn{{{Text: b.t("btn.cancel"), Data: "tx:x"}}}, false)
			return
		}
		s.Amount = n
		b.askNote(chat)
		return
	}
	if s.Step == "tx_note" {
		s.Note = clip(text, 80)
		b.askConfirm(ctx, chat, 0)
		return
	}

	switch {
	case strings.HasPrefix(low, "/expense") || b.isBtn(text, "kb.expense"):
		b.startTx(ctx, chat, 0, "expense")
	case strings.HasPrefix(low, "/income") || b.isBtn(text, "kb.income"):
		b.startTx(ctx, chat, 0, "income")
	case strings.HasPrefix(low, "/transfer") || b.isBtn(text, "kb.transfer"):
		b.startTx(ctx, chat, 0, "transfer")
	case b.isBtn(text, "kb.overview"):
		b.showOverview(ctx, chat, 0)
	case b.isBtn(text, "kb.accounts"):
		b.showAccounts(ctx, chat, 0)
	case b.isBtn(text, "kb.assets"):
		b.showAssets(ctx, chat, 0)
	case b.isBtn(text, "kb.debts"):
		b.showDebts(ctx, chat, 0)
	case b.isBtn(text, "kb.projects"):
		b.showProjects(ctx, chat, 0)
	default:
		_ = b.sendTo(chat, b.t("help.unknown"), navRow(b), true)
	}
}

func (b *Bot) onCallback(ctx context.Context, chat int64, msgID int, data string) {
	switch {
	case data == "go:home":
		b.clear(chat)
		_ = b.sendHome(ctx, chat, msgID, false)
	case data == "go:ov":
		b.showOverview(ctx, chat, msgID)
	case data == "go:ac":
		b.showAccounts(ctx, chat, msgID)
	case data == "go:as":
		b.showAssets(ctx, chat, msgID)
	case data == "go:de":
		b.showDebts(ctx, chat, msgID)
	case data == "go:pr":
		b.showProjects(ctx, chat, msgID)
	case data == "go:re":
		b.showRecent(ctx, chat, msgID)
	case data == "go:lang":
		b.showLang(chat, msgID)
	case strings.HasPrefix(data, "lang:"):
		code := strings.TrimPrefix(data, "lang:")
		if code == "fa" || code == "en" || code == "de" {
			_ = b.store.SetSetting(ctx, "telegram_ui_lang", code)
		}
		_ = b.sendHome(ctx, chat, 0, true)
	case data == "tx:expense" || data == "tx:income" || data == "tx:transfer":
		b.startTx(ctx, chat, msgID, strings.TrimPrefix(data, "tx:"))
	case strings.HasPrefix(data, "tx:a:"):
		b.pickAccount(ctx, chat, msgID, strings.TrimPrefix(data, "tx:a:"), false)
	case strings.HasPrefix(data, "tx:t:"):
		b.pickAccount(ctx, chat, msgID, strings.TrimPrefix(data, "tx:t:"), true)
	case strings.HasPrefix(data, "tx:c:"):
		b.pickCategory(ctx, chat, msgID, strings.TrimPrefix(data, "tx:c:"))
	case data == "tx:skipc":
		b.pickCategory(ctx, chat, msgID, "")
	case data == "tx:skipn":
		s := b.get(chat)
		s.Note = ""
		b.askConfirm(ctx, chat, msgID)
	case data == "tx:ok":
		b.commitTx(ctx, chat, msgID)
	case data == "tx:x":
		b.clear(chat)
		b.show(chat, msgID, b.t("tx.cancelled"), navRow(b))
	case strings.HasPrefix(data, "d:p:"):
		b.startPay(ctx, chat, msgID, strings.TrimPrefix(data, "d:p:"))
	case strings.HasPrefix(data, "d:a:"):
		b.pickPayAccount(ctx, chat, msgID, strings.TrimPrefix(data, "d:a:"))
	case data == "d:ok":
		b.commitPay(ctx, chat, msgID)
	}
}

func (b *Bot) sendHome(ctx context.Context, chat int64, msgID int, withKB bool) error {
	text := b.t("home.title") + "\n\n" + b.t("home.body")
	if withKB {
		_ = b.sendTo(chat, b.t("home.title"), nil, true)
		return b.sendTo(chat, text, navRow(b), false)
	}
	if msgID == 0 {
		return b.sendTo(chat, text, navRow(b), false)
	}
	b.show(chat, msgID, text, navRow(b))
	return nil
}

func (b *Bot) showLang(chat int64, msgID int) {
	b.show(chat, msgID, b.t("lang.title"), [][]btn{
		{{Text: "فارسی", Data: "lang:fa"}, {Text: "English", Data: "lang:en"}, {Text: "Deutsch", Data: "lang:de"}},
		backRow(b),
	})
}

func (b *Bot) showOverview(ctx context.Context, chat int64, msgID int) {
	d, err := b.store.Dashboard(ctx)
	if err != nil {
		b.show(chat, msgID, err.Error(), navRow(b))
		return
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "📊 %s\n\n", b.t("overview"))
	fmt.Fprintf(&sb, "💠 %s: %s\n", b.t("net"), toman(d.NetWorth))
	fmt.Fprintf(&sb, "💳 %s: %s\n", b.t("liquid"), toman(d.Liquid))
	fmt.Fprintf(&sb, "💎 %s: %s\n", b.t("assets"), toman(d.AssetsTotal))
	fmt.Fprintf(&sb, "📉 %s: %s\n", b.t("debts"), toman(d.DebtsRemaining))
	fmt.Fprintf(&sb, "🟢 %s: %s\n", b.t("monthIn"), toman(d.MonthlyIncome))
	fmt.Fprintf(&sb, "🔴 %s: %s\n", b.t("monthOut"), toman(d.MonthlyExpense))
	if len(d.Upcoming) > 0 {
		u := d.Upcoming[0]
		fmt.Fprintf(&sb, "\n⏰ %s: %s — %s (%s)", b.t("nextDue"), u.DebtName, toman(u.Amount), u.DueDate.Format("02 Jan"))
	}
	b.show(chat, msgID, sb.String(), navRow(b))
}

func (b *Bot) showAccounts(ctx context.Context, chat int64, msgID int) {
	items, err := b.store.ListAccounts(ctx)
	if err != nil || len(items) == 0 {
		b.show(chat, msgID, b.t("empty.acc"), [][]btn{backRow(b)})
		return
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "💳 %s\n", b.t("accounts"))
	var total int64
	for _, a := range items {
		total += a.Balance
		fmt.Fprintf(&sb, "\n• %s\n  %s", a.Name, toman(a.Balance))
		if a.BankName != "" {
			fmt.Fprintf(&sb, " · %s", a.BankName)
		}
	}
	fmt.Fprintf(&sb, "\n\nΣ %s", toman(total))
	b.show(chat, msgID, sb.String(), [][]btn{
		{{Text: b.t("btn.expense"), Data: "tx:expense"}, {Text: b.t("btn.income"), Data: "tx:income"}},
		backRow(b),
	})
}

func (b *Bot) showAssets(ctx context.Context, chat int64, msgID int) {
	items, err := b.store.ListAssets(ctx)
	if err != nil || len(items) == 0 {
		b.show(chat, msgID, b.t("empty.asset"), [][]btn{backRow(b)})
		return
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "💎 %s\n", b.t("assetsT"))
	var total int64
	for _, a := range items {
		total += a.Value
		fmt.Fprintf(&sb, "\n• %s\n  %s", a.Name, toman(a.Value))
		if a.Quantity != 0 {
			fmt.Fprintf(&sb, " · %g %s", a.Quantity, a.Unit)
		}
	}
	fmt.Fprintf(&sb, "\n\nΣ %s", toman(total))
	b.show(chat, msgID, sb.String(), [][]btn{backRow(b)})
}

func (b *Bot) showDebts(ctx context.Context, chat int64, msgID int) {
	items, err := b.store.ListDebts(ctx)
	if err != nil || len(items) == 0 {
		b.show(chat, msgID, b.t("empty.debt"), [][]btn{backRow(b)})
		return
	}
	s := b.get(chat)
	s.Opts = nil
	var sb strings.Builder
	fmt.Fprintf(&sb, "📉 %s\n", b.t("debtsT"))
	rows := [][]btn{}
	for i, d := range items {
		st := b.t("active")
		if d.Status == "paid" {
			st = b.t("paid")
		}
		fmt.Fprintf(&sb, "\n• %s (%s)\n  %s", d.Name, st, toman(d.Remaining))
		if d.CommissionAmount > 0 {
			fmt.Fprintf(&sb, "\n  💳 fee %s · net %s", toman(d.CommissionAmount), toman(d.NetReceived))
		}
		if d.Creditor != "" {
			fmt.Fprintf(&sb, " · %s", d.Creditor)
		}
		if d.NextDue != nil {
			fmt.Fprintf(&sb, "\n  ⏰ %s", d.NextDue.Format("02 Jan 2006"))
		}
		if d.Status != "paid" && d.Remaining > 0 {
			s.Opts = append(s.Opts, d.ID)
			label := clip(b.t("btn.pay")+" · "+d.Name, 40)
			rows = append(rows, []btn{{Text: label, Data: "d:p:" + strconv.Itoa(len(s.Opts)-1)}})
			_ = i
		}
	}
	rows = append(rows, backRow(b))
	b.show(chat, msgID, sb.String(), rows)
}

func (b *Bot) showProjects(ctx context.Context, chat int64, msgID int) {
	items, err := b.store.ListProjects(ctx)
	if err != nil || len(items) == 0 {
		b.show(chat, msgID, b.t("empty.proj"), [][]btn{backRow(b)})
		return
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "📁 %s\n", b.t("projects"))
	for _, p := range items {
		pct := 0.0
		if p.TargetAmount > 0 {
			pct = float64(p.CurrentAmount) / float64(p.TargetAmount) * 100
		}
		fmt.Fprintf(&sb, "\n• %s\n  %s / %s (%.0f%%)", p.Name, toman(p.CurrentAmount), toman(p.TargetAmount), pct)
	}
	b.show(chat, msgID, sb.String(), [][]btn{backRow(b)})
}

func (b *Bot) showRecent(ctx context.Context, chat int64, msgID int) {
	items, err := b.store.ListTransactions(ctx, 8, "", "", "")
	if err != nil || len(items) == 0 {
		b.show(chat, msgID, b.t("empty.tx"), [][]btn{backRow(b)})
		return
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "📜 %s\n", b.t("recent"))
	for _, t := range items {
		icon := "•"
		sign := ""
		switch t.Type {
		case "income":
			icon, sign = "🟢", "+"
		case "expense", "contribution":
			icon, sign = "🔴", "−"
		case "transfer":
			icon = "↔️"
		}
		line := fmt.Sprintf("\n%s %s%s", icon, sign, toman(t.Amount))
		if t.Description != "" {
			line += " · " + clip(t.Description, 28)
		} else if t.Category != "" {
			line += " · " + t.Category
		}
		if t.AccountName != "" {
			line += "\n   " + t.AccountName
		}
		sb.WriteString(line)
	}
	b.show(chat, msgID, sb.String(), [][]btn{
		{{Text: b.t("btn.expense"), Data: "tx:expense"}, {Text: b.t("btn.income"), Data: "tx:income"}},
		backRow(b),
	})
}

func chunkBtns(items []btn, n int) [][]btn {
	out := [][]btn{}
	for i := 0; i < len(items); i += n {
		j := i + n
		if j > len(items) {
			j = len(items)
		}
		out = append(out, items[i:j])
	}
	return out
}

func accButtons(accounts []models.Account, prefix string) ([]string, [][]btn) {
	ids := make([]string, 0, len(accounts))
	btns := make([]btn, 0, len(accounts))
	for i, a := range accounts {
		ids = append(ids, a.ID)
		btns = append(btns, btn{
			Text: clip(fmt.Sprintf("%s · %s", a.Name, money(a.Balance)), 40),
			Data: prefix + strconv.Itoa(i),
		})
	}
	return ids, chunkBtns(btns, 1)
}
