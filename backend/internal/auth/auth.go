package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

const (
	CookieName = "pvmoney_session"
	SessionTTL = 30 * 24 * 3600
)

var usernameRe = regexp.MustCompile(`^[\p{L}\p{N}_.]{3,24}$`)

func Hash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func HashOTP(challengeID, otp string) string {
	return Hash(challengeID + ":" + otp)
}

func RandomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func RandomLink() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func RandomOTP() (string, error) {
	var n uint32
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	n = uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
	return fmt.Sprintf("%06d", n%1000000), nil
}

func NormalizeUsername(s string) string {
	return strings.TrimSpace(s)
}

func ValidUsername(s string) bool {
	s = NormalizeUsername(s)
	if strings.ContainsAny(s, " \t\n") {
		return false
	}
	return usernameRe.MatchString(s)
}

func FoldDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= '۰' && r <= '۹':
			b.WriteByte(byte(r - '۰' + '0'))
		case r >= '٠' && r <= '٩':
			b.WriteByte(byte(r - '٠' + '0'))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func NormalizePhone(raw string) (string, error) {
	raw = strings.TrimSpace(FoldDigits(raw))
	var d strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			d.WriteRune(r)
		}
	}
	digits := d.String()
	if strings.HasPrefix(digits, "00") {
		digits = digits[2:]
	}
	switch {
	case strings.HasPrefix(digits, "98") && len(digits) == 12:
		return "+" + digits, nil
	case strings.HasPrefix(digits, "0") && len(digits) == 11 && digits[1] == '9':
		return "+98" + digits[1:], nil
	case len(digits) == 10 && digits[0] == '9':
		return "+98" + digits, nil
	case len(digits) >= 10 && len(digits) <= 15:
		return "+" + digits, nil
	default:
		return "", fmt.Errorf("invalid_phone")
	}
}

func MaskPhone(phone string) string {
	if len(phone) < 6 {
		return phone
	}
	return phone[:len(phone)-4] + "··" + phone[len(phone)-2:]
}

func Bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if c, err := r.Cookie(CookieName); err == nil {
		return c.Value
	}
	return ""
}

func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   SessionTTL,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
