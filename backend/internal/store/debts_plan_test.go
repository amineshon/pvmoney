package store

import (
	"testing"
	"time"
)

func TestSplitAmountsRemainder(t *testing.T) {
	got, err := splitAmounts(30_000_000, 11_500_000, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{11_500_000, 11_500_000, 7_000_000}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSplitAmountsEven(t *testing.T) {
	got, err := splitAmounts(30_000_000, 10_000_000, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != 10_000_000 || got[2] != 10_000_000 {
		t.Fatalf("got %v", got)
	}
}

func TestJalaliRoundTripNowruz(t *testing.T) {
	jy, jm, jd := gregorianToJalali(2025, 11, 18)
	if jy != 1404 || jm != 8 || jd != 27 {
		t.Fatalf("27 Aban 1404: got %d/%d/%d", jy, jm, jd)
	}
	gy, gm, gd := jalaliToGregorian(1404, 8, 27)
	if gy != 2025 || gm != 11 || gd != 18 {
		t.Fatalf("back to gregorian: %d-%d-%d", gy, gm, gd)
	}
}

func TestInstallmentKeepsJalaliDay(t *testing.T) {
	first := civilDate(2025, 11, 18) // 27 Aban 1404
	dates := datesFromFirst(first, 27, 4)
	want := []string{"2025-11-18", "2025-12-18", "2026-01-17", "2026-02-16"} // 27 Aban, Azar, Dey, Bahman
	if len(dates) != len(want) {
		t.Fatalf("len=%d", len(dates))
	}
	for i, w := range want {
		got := dates[i].UTC().Format("2006-01-02")
		if got != w {
			y, m, d := dates[i].UTC().Date()
			jy, jm, jd := gregorianToJalali(y, int(m), d)
			t.Fatalf("inst %d: got %s (j %d/%d/%d) want %s", i+1, got, jy, jm, jd, w)
		}
		y, m, d := dates[i].UTC().Date()
		_, _, jd := gregorianToJalali(y, int(m), d)
		if jd != 27 {
			t.Fatalf("inst %d jalali day=%d want 27", i+1, jd)
		}
	}
}

func TestInstallmentClampsJalaliMonthEnd(t *testing.T) {
	first := civilDate(2025, 9, 22) // 31 Shahrivar 1404
	dates := datesFromFirst(first, 31, 2)
	got0 := dates[0].UTC().Format("2006-01-02")
	got1 := dates[1].UTC().Format("2006-01-02")
	if got0 != "2025-09-22" {
		t.Fatalf("first=%s", got0)
	}
	if got1 != "2025-10-22" { // 30 Mehr
		t.Fatalf("second=%s want 2025-10-22 (30 Mehr)", got1)
	}
}

func TestJalaliMonthRangeMehr1405(t *testing.T) {
	start, end := jalaliMonthRange(time.Date(2026, 9, 28, 12, 0, 0, 0, tehran()))
	if start != "2026-09-23" || end != "2026-10-23" {
		t.Fatalf("range %s .. %s", start, end)
	}
}

func TestParseDateDoesNotShiftDay(t *testing.T) {
	s := "2025-11-18"
	v, err := parseDate(&s)
	if err != nil {
		t.Fatal(err)
	}
	tme := v.(time.Time)
	if tme.UTC().Format("2006-01-02") != "2025-11-18" {
		t.Fatalf("got %s", tme.UTC())
	}
}

