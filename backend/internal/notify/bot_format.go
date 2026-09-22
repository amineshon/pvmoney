package notify

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"pvmoney/internal/models"
)

func money(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	return sign + out
}

func toman(n int64) string {
	return money(n) + " Toman"
}

func enDigits(s string) string {
	r := strings.NewReplacer(
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
		"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
		"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4",
		"٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
		"،", ".", "٫", ".",
	)
	return r.Replace(s)
}

func parseAmount(raw string) (int64, error) {
	s := strings.TrimSpace(strings.ToLower(enDigits(raw)))
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	mul := int64(1)
	switch {
	case strings.Contains(s, "میلیارد") || strings.HasSuffix(s, "b") || strings.Contains(s, "billion"):
		mul = 1_000_000_000
	case strings.Contains(s, "میلیون") || strings.HasSuffix(s, "m") || strings.Contains(s, "million") || strings.Contains(s, "mio"):
		mul = 1_000_000
	case strings.Contains(s, "هزار") || strings.HasSuffix(s, "k") || strings.Contains(s, "thousand"):
		mul = 1_000
	}
	var b strings.Builder
	dot := false
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		if r == '.' && !dot {
			b.WriteRune(r)
			dot = true
		}
	}
	num := b.String()
	if num == "" || num == "." {
		return 0, fmt.Errorf("no number")
	}
	if strings.Contains(num, ".") {
		f, err := strconv.ParseFloat(num, 64)
		if err != nil {
			return 0, err
		}
		n := int64(f * float64(mul))
		if n <= 0 || n > models.MaxMoney {
			return 0, fmt.Errorf("amount too large")
		}
		return n, nil
	}
	n, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return 0, err
	}
	if mul > 1 && n > models.MaxMoney/mul {
		return 0, fmt.Errorf("amount too large")
	}
	out := n * mul
	if out <= 0 || out > models.MaxMoney {
		return 0, fmt.Errorf("amount too large")
	}
	return out, nil
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

func formatReminder(it models.Installment, kind string, loc *time.Location) string {
	due := it.DueDate.In(loc).Format("02 Jan 2006")
	dueDE := it.DueDate.In(loc).Format("02.01.2006")
	whenEN, whenDE := "Payment due today", "Zahlung heute fällig"
	switch kind {
	case "d20":
		whenEN, whenDE = "Payment in 20 days", "Zahlung in 20 Tagen"
	case "d7":
		whenEN, whenDE = "Payment in 7 days", "Zahlung in 7 Tagen"
	case "d3":
		whenEN, whenDE = "Payment in 3 days", "Zahlung in 3 Tagen"
	}
	cred := strings.TrimSpace(it.Creditor)
	if cred == "" {
		cred = "—"
	}
	return fmt.Sprintf(
		"⚠️ PVMoney — %s\n\nDebt: %s\nTo: %s\nAmount: %s\nDue: %s\n\n————————\n\n⚠️ PVMoney — %s\n\nSchuld: %s\nAn: %s\nBetrag: %s\nFällig: %s",
		whenEN, it.DebtName, cred, toman(it.Amount), due,
		whenDE, it.DebtName, cred, toman(it.Amount), dueDE,
	)
}

func navRow(b *Bot) [][]btn {
	return [][]btn{
		{{Text: b.t("btn.overview"), Data: "go:ov"}},
		{{Text: b.t("btn.accounts"), Data: "go:ac"}, {Text: b.t("btn.assets"), Data: "go:as"}},
		{{Text: b.t("btn.debts"), Data: "go:de"}, {Text: b.t("btn.projects"), Data: "go:pr"}},
		{{Text: b.t("btn.recent"), Data: "go:re"}},
		{{Text: b.t("btn.expense"), Data: "tx:expense"}, {Text: b.t("btn.income"), Data: "tx:income"}},
		{{Text: b.t("btn.transfer"), Data: "tx:transfer"}},
		{{Text: b.t("btn.lang"), Data: "go:lang"}},
	}
}

func backRow(b *Bot) []btn {
	return []btn{{Text: b.t("btn.back"), Data: "go:home"}}
}
