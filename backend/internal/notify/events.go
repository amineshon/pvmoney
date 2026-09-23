package notify

import (
	"context"
	"fmt"

	"pvmoney/internal/models"
)

type Event struct {
	Icon string
	EN   string
	DE   string
}

func (b *Bot) OnAppEvent(ctx context.Context, ev Event) {
	if ev.EN == "" {
		return
	}
	text := fmt.Sprintf("%s %s\n\n————————\n\n%s %s", ev.Icon, ev.EN, ev.Icon, ev.DE)
	_ = b.Send(ctx, text)
}

func EvTx(t models.Transaction) Event {
	switch t.Type {
	case "income":
		return Event{Icon: "🟢", EN: fmt.Sprintf("Income recorded\n+%s\n%s", toman(t.Amount), txBits(t)), DE: fmt.Sprintf("Einnahme gebucht\n+%s\n%s", toman(t.Amount), txBits(t))}
	case "expense", "contribution":
		return Event{Icon: "🔴", EN: fmt.Sprintf("Expense recorded\n−%s\n%s", toman(t.Amount), txBits(t)), DE: fmt.Sprintf("Ausgabe gebucht\n−%s\n%s", toman(t.Amount), txBits(t))}
	case "transfer":
		return Event{Icon: "↔️", EN: fmt.Sprintf("Transfer recorded\n%s\n%s", toman(t.Amount), txBits(t)), DE: fmt.Sprintf("Überweisung gebucht\n%s\n%s", toman(t.Amount), txBits(t))}
	default:
		return Event{Icon: "✏️", EN: fmt.Sprintf("Transaction\n%s\n%s", toman(t.Amount), txBits(t)), DE: fmt.Sprintf("Buchung\n%s\n%s", toman(t.Amount), txBits(t))}
	}
}

func txBits(t models.Transaction) string {
	s := ""
	if t.AccountName != "" {
		s = t.AccountName
		if t.ToName != "" {
			s += " → " + t.ToName
		}
	}
	if t.Category != "" {
		if s != "" {
			s += " · "
		}
		s += t.Category
	}
	if t.Description != "" {
		if s != "" {
			s += "\n"
		}
		s += t.Description
	}
	if t.ProjectName != "" {
		s += "\n📁 " + t.ProjectName
	}
	return s
}

func EvAccountNew(a models.Account) Event {
	return Event{Icon: "💳", EN: fmt.Sprintf("Account added\n%s\nBalance: %s", a.Name, toman(a.Balance)), DE: fmt.Sprintf("Konto hinzugefügt\n%s\nSaldo: %s", a.Name, toman(a.Balance))}
}

func EvAccountUpd(a models.Account) Event {
	return Event{Icon: "💳", EN: fmt.Sprintf("Account updated\n%s", a.Name), DE: fmt.Sprintf("Konto aktualisiert\n%s", a.Name)}
}

func EvAccountDel(name string) Event {
	return Event{Icon: "🗑", EN: "Account deleted\n" + name, DE: "Konto gelöscht\n" + name}
}

func EvAdjust(a models.Account) Event {
	return Event{Icon: "⚖️", EN: fmt.Sprintf("Balance adjusted\n%s\nNow: %s", a.Name, toman(a.Balance)), DE: fmt.Sprintf("Saldo angepasst\n%s\nJetzt: %s", a.Name, toman(a.Balance))}
}

func EvAssetNew(a models.Asset) Event {
	return Event{Icon: "💎", EN: fmt.Sprintf("Asset added\n%s\n%s", a.Name, toman(a.Value)), DE: fmt.Sprintf("Vermögen hinzugefügt\n%s\n%s", a.Name, toman(a.Value))}
}

func EvAssetUpd(a models.Asset) Event {
	return Event{Icon: "💎", EN: fmt.Sprintf("Asset updated\n%s\n%s", a.Name, toman(a.Value)), DE: fmt.Sprintf("Vermögen aktualisiert\n%s\n%s", a.Name, toman(a.Value))}
}

func EvAssetDel(name string) Event {
	return Event{Icon: "🗑", EN: "Asset deleted\n" + name, DE: "Vermögen gelöscht\n" + name}
}

func EvProjectNew(p models.Project) Event {
	return Event{Icon: "📁", EN: fmt.Sprintf("Project created\n%s\nBudget: %s", p.Name, toman(p.TargetAmount)), DE: fmt.Sprintf("Projekt erstellt\n%s\nBudget: %s", p.Name, toman(p.TargetAmount))}
}

func EvProjectUpd(p models.Project) Event {
	return Event{Icon: "📁", EN: fmt.Sprintf("Project updated\n%s", p.Name), DE: fmt.Sprintf("Projekt aktualisiert\n%s", p.Name)}
}

func EvProjectDel(name string) Event {
	return Event{Icon: "🗑", EN: "Project deleted\n" + name, DE: "Projekt gelöscht\n" + name}
}

func EvProjectPay(name string, amount int64) Event {
	return Event{Icon: "🔴", EN: fmt.Sprintf("Project payment\n%s\n−%s", name, toman(amount)), DE: fmt.Sprintf("Projektzahlung\n%s\n−%s", name, toman(amount))}
}

func EvRefund(name string, amount int64) Event {
	return Event{Icon: "↩️", EN: fmt.Sprintf("Project refund\n%s\n+%s", name, toman(amount)), DE: fmt.Sprintf("Projektrückzahlung\n%s\n+%s", name, toman(amount))}
}

func EvConvert(name string, value int64) Event {
	return Event{Icon: "🏠", EN: fmt.Sprintf("Project saved as asset\n%s\n%s", name, toman(value)), DE: fmt.Sprintf("Projekt als Vermögen gespeichert\n%s\n%s", name, toman(value))}
}

func EvDebtNew(d models.Debt) Event {
	en := fmt.Sprintf("Debt added\n%s\nRemaining: %s", d.Name, toman(d.Remaining))
	de := fmt.Sprintf("Schuld hinzugefügt\n%s\nRest: %s", d.Name, toman(d.Remaining))
	if d.CommissionAmount > 0 {
		en += fmt.Sprintf("\nFee: %s\nNet received: %s", toman(d.CommissionAmount), toman(d.NetReceived))
		de += fmt.Sprintf("\nGebühr: %s\nNetto: %s", toman(d.CommissionAmount), toman(d.NetReceived))
	}
	return Event{Icon: "📉", EN: en, DE: de}
}

func EvDebtUpd(d models.Debt) Event {
	return Event{Icon: "📉", EN: fmt.Sprintf("Debt updated\n%s", d.Name), DE: fmt.Sprintf("Schuld aktualisiert\n%s", d.Name)}
}

func EvDebtDel(name string) Event {
	return Event{Icon: "🗑", EN: "Debt deleted\n" + name, DE: "Schuld gelöscht\n" + name}
}

func EvDebtPay(name string, amount, remaining int64) Event {
	return Event{Icon: "💸", EN: fmt.Sprintf("Debt payment\n%s\n−%s\nRemaining: %s", name, toman(amount), toman(remaining)), DE: fmt.Sprintf("Schuldzahlung\n%s\n−%s\nRest: %s", name, toman(amount), toman(remaining))}
}

func EvTxDeleted() Event {
	return Event{Icon: "🗑", EN: "Transaction deleted and balance reversed.", DE: "Buchung gelöscht, Saldo rückgängig gemacht."}
}
