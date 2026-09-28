package store

import "time"

func div(a, b int) int { return a / b }

func gregorianToJalali(gy, gm, gd int) (jy, jm, jd int) {
	gdm := [12]int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	gy2 := gy
	if gm > 2 {
		gy2 = gy + 1
	}
	days := 355666 + 365*gy + div(gy2+3, 4) - div(gy2+99, 100) + div(gy2+399, 400) + gd + gdm[gm-1]
	jy = -1595 + 33*div(days, 12053)
	days %= 12053
	jy += 4 * div(days, 1461)
	days %= 1461
	if days > 365 {
		jy += div(days-1, 365)
		days = (days - 1) % 365
	}
	if days < 186 {
		jm = 1 + div(days, 31)
		jd = 1 + (days % 31)
	} else {
		jm = 7 + div(days-186, 30)
		jd = 1 + ((days - 186) % 30)
	}
	return
}

func jalaliToGregorian(jy, jm, jd int) (gy, gm, gd int) {
	jy += 1595
	days := -355668 + 365*jy + div(jy, 33)*8 + div((jy%33)+3, 4) + jd
	if jm < 7 {
		days += (jm - 1) * 31
	} else {
		days += (jm-7)*30 + 186
	}
	gy = 400 * div(days, 146097)
	days %= 146097
	if days > 36524 {
		days--
		gy += 100 * div(days, 36524)
		days %= 36524
		if days >= 365 {
			days++
		}
	}
	gy += 4 * div(days, 1461)
	days %= 1461
	if days > 365 {
		gy += div(days-1, 365)
		days = (days - 1) % 365
	}
	gd = days + 1
	leap := gy%4 == 0 && gy%100 != 0 || gy%400 == 0
	sal := [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if leap {
		sal[2] = 29
	}
	gm = 0
	for gm < 12 && gd > sal[gm] {
		gd -= sal[gm]
		gm++
	}
	return gy, gm, gd
}

func jalaliLeap(jy int) bool {
	r := ((jy % 33) + 33) % 33
	switch r {
	case 1, 5, 9, 13, 17, 22, 26, 30:
		return true
	default:
		return false
	}
}

func jalaliMonthLen(jy, jm int) int {
	if jm <= 6 {
		return 31
	}
	if jm <= 11 {
		return 30
	}
	if jalaliLeap(jy) {
		return 30
	}
	return 29
}

func civilDate(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.UTC)
}

func dateYMD(t time.Time) (y, m, d int) {
	t = t.In(tehran())
	return t.Year(), int(t.Month()), t.Day()
}

func jalaliDayOf(t time.Time) int {
	y, m, d := dateYMD(t)
	_, _, jd := gregorianToJalali(y, m, d)
	return jd
}

func addJalaliMonths(t time.Time, n int) time.Time {
	y, m, d := dateYMD(t)
	jy, jm, jd := gregorianToJalali(y, m, d)
	idx := jy*12 + (jm - 1) + n
	ny := idx / 12
	nm := idx%12 + 1
	if nm <= 0 {
		nm += 12
		ny--
	}
	last := jalaliMonthLen(ny, nm)
	if jd > last {
		jd = last
	}
	gy, gm, gd := jalaliToGregorian(ny, nm, jd)
	return civilDate(gy, gm, gd)
}
