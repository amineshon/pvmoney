package notify

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"pvmoney/internal/auth"
)

func (b *Bot) handleAuthStart(ctx context.Context, chat int64, text string) bool {
	low := strings.ToLower(strings.TrimSpace(text))
	if !strings.HasPrefix(low, "/start") {
		return false
	}
	payload := startPayload(text)
	if payload == "" {
		return false
	}
	ch, err := b.store.ChallengeByLinkToken(ctx, payload)
	if err != nil {
		_ = b.sendTo(chat, tr("fa", "auth.badLink"), nil, false)
		return true
	}
	id := strconv.FormatInt(chat, 10)
	if err := b.store.SetChallengeStart(ctx, ch.ID, id); err != nil {
		log.Printf("telegram link: %v", err)
		return true
	}
	_ = b.sendContactRequest(chat, ch.Lang, ch.Username)
	return true
}

func (b *Bot) handleContact(ctx context.Context, chat int64, msg *tgMessage) {
	if msg.Contact == nil {
		return
	}
	if msg.From != nil && msg.Contact.UserID != 0 && msg.Contact.UserID != msg.From.ID {
		_ = b.sendTo(chat, tr("fa", "auth.ownPhone"), nil, false)
		return
	}
	id := strconv.FormatInt(chat, 10)
	ch, err := b.store.ChallengeByChatPending(ctx, id)
	if err != nil {
		_ = b.sendTo(chat, tr("fa", "auth.needApp"), nil, false)
		return
	}
	got, err := auth.NormalizePhone(msg.Contact.PhoneNumber)
	if err != nil {
		_ = b.store.SetChallengeMismatch(ctx, ch.ID, msg.Contact.PhoneNumber)
		_ = b.sendTo(chat, tr(ch.Lang, "auth.phoneBad"), nil, false)
		return
	}
	if got != ch.TelegramPhone {
		_ = b.store.SetChallengeMismatch(ctx, ch.ID, got)
		_ = b.sendTo(chat, tr(ch.Lang, "auth.phoneMismatch"), nil, false)
		return
	}
	otp, err := auth.RandomOTP()
	if err != nil {
		return
	}
	exp := time.Now().Add(5 * time.Minute)
	if err := b.store.SetChallengeOTP(ctx, ch.ID, id, got, auth.HashOTP(ch.ID, otp), exp); err != nil {
		log.Printf("telegram otp store: %v", err)
		return
	}
	_ = b.sendTo(chat, otpText(ch.Lang, otp, ch.Username), nil, false)
}

func otpText(lang, otp, username string) string {
	switch lang {
	case "de":
		return "Nummer bestätigt. PVMoney-Code für " + username + ":\n\n" + otp + "\n\nGültig 5 Minuten. Trage ihn in der App ein."
	case "en":
		return "Number confirmed. PVMoney code for " + username + ":\n\n" + otp + "\n\nValid for 5 minutes. Enter it in the app."
	default:
		return "شماره تایید شد. کد پی‌وی‌مانی برای " + username + ":\n\n" + otp + "\n\nتا ۵ دقیقه معتبر است. همان را در سایت وارد کن."
	}
}

func (b *Bot) sendContactRequest(chat int64, lang, username string) error {
	text := strings.ReplaceAll(tr(lang, "auth.askContact"), "{name}", username)
	markup := map[string]any{
		"keyboard": [][]map[string]any{{
			{"text": tr(lang, "auth.sharePhone"), "request_contact": true},
		}},
		"resize_keyboard":   true,
		"one_time_keyboard": true,
	}
	return b.sendMarkup(chat, text, markup)
}

func startPayload(text string) string {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) < 2 {
		return ""
	}
	p := parts[len(parts)-1]
	if strings.HasPrefix(p, "/start") {
		return ""
	}
	return p
}
