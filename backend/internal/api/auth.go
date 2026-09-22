package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"pvmoney/internal/auth"
	"pvmoney/internal/models"
	"pvmoney/internal/store"

	"github.com/go-chi/chi/v5"
)

type ctxKey int

const userCtxKey ctxKey = 1

func (s *Server) withUser(ctx context.Context, u models.User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

func UserFrom(ctx context.Context) (models.User, bool) {
	u, ok := ctx.Value(userCtxKey).(models.User)
	return u, ok
}

func (s *Server) userFromRequest(r *http.Request) (models.User, error) {
	tok := auth.Bearer(r)
	if tok == "" {
		return models.User{}, store.ErrUnauthorized
	}
	return s.store.UserBySession(r.Context(), auth.Hash(tok))
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := s.userFromRequest(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r.WithContext(s.withUser(r.Context(), u)))
	})
}

type challengeView struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Purpose     string `json:"purpose"`
	BotLink     string `json:"bot_link,omitempty"`
	BotUsername string `json:"bot_username,omitempty"`
	PhoneMasked string `json:"phone_masked,omitempty"`
	ExpiresIn   int    `json:"expires_in"`
}

func (s *Server) viewChallenge(c models.AuthChallenge) challengeView {
	bot, _ := s.store.GetSetting(context.Background(), "telegram_bot_username")
	v := challengeView{
		ID:          c.ID,
		Status:      c.Status,
		Purpose:     c.Purpose,
		BotUsername: bot,
		PhoneMasked: auth.MaskPhone(c.TelegramPhone),
		ExpiresIn:   int(time.Until(c.ExpiresAt).Seconds()),
	}
	if bot != "" && c.LinkToken != "" && (c.Purpose == "register" || c.Purpose == "reconnect") {
		v.BotLink = "https://t.me/" + bot + "?start=" + c.LinkToken
	}
	if v.ExpiresIn < 0 {
		v.ExpiresIn = 0
	}
	return v
}

func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	has, err := s.store.HasUsers(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	bot, _ := s.store.GetSetting(r.Context(), "telegram_bot_username")
	mode := "register"
	if has {
		mode = "login"
	}
	out := map[string]any{
		"authenticated": false,
		"mode":          mode,
		"has_users":     has,
		"bot_username":  bot,
		"bot_ready":     s.bot != nil && bot != "",
	}
	if u, err := s.userFromRequest(r); err == nil {
		out["authenticated"] = true
		out["user"] = publicUser(u)
		out["mode"] = "login"
	}
	writeJSON(w, http.StatusOK, out)
}

func publicUser(u models.User) map[string]any {
	return map[string]any{
		"id":                 u.ID,
		"username":           u.Username,
		"telegram_phone":     auth.MaskPhone(u.TelegramPhone),
		"telegram_verified":  u.TelegramVerified,
		"telegram_connected": u.TelegramChatID != "" && u.TelegramVerified,
	}
}

func (s *Server) registerStart(w http.ResponseWriter, r *http.Request) {
	if s.bot == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "telegram_disabled"})
		return
	}
	var in struct {
		Username string `json:"username"`
		Phone    string `json:"phone"`
		Lang     string `json:"lang"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	username := auth.NormalizeUsername(in.Username)
	if !auth.ValidUsername(username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_username"})
		return
	}
	phone, err := auth.NormalizePhone(in.Phone)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_phone"})
		return
	}
	if _, err := s.store.UserByUsername(r.Context(), username); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "username_taken"})
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		writeErr(w, err)
		return
	}
	if _, err := s.store.UserByPhone(r.Context(), phone); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "phone_taken"})
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		writeErr(w, err)
		return
	}
	link, err := auth.RandomLink()
	if err != nil {
		writeErr(w, err)
		return
	}
	lang := in.Lang
	if lang != "en" && lang != "de" && lang != "fa" {
		lang = "fa"
	}
	ch, err := s.store.CreateChallenge(r.Context(), models.AuthChallenge{
		Purpose:       "register",
		Username:      username,
		TelegramPhone: phone,
		Lang:          lang,
		LinkToken:     link,
		Status:        "pending_start",
		ExpiresAt:     time.Now().Add(20 * time.Minute),
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.viewChallenge(ch))
}

func (s *Server) challengeStatus(w http.ResponseWriter, r *http.Request) {
	ch, err := s.store.ChallengeByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	if time.Now().After(ch.ExpiresAt) || ch.Status == "consumed" {
		writeJSON(w, http.StatusGone, map[string]string{"error": "expired"})
		return
	}
	writeJSON(w, http.StatusOK, s.viewChallenge(ch))
}

func (s *Server) loginStart(w http.ResponseWriter, r *http.Request) {
	if s.bot == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "telegram_disabled"})
		return
	}
	var in struct {
		Username string `json:"username"`
		Lang     string `json:"lang"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	username := auth.NormalizeUsername(in.Username)
	u, err := s.store.UserByUsername(r.Context(), username)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	if u.TelegramChatID == "" || !u.TelegramVerified {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "telegram_disconnected"})
		return
	}
	lang := in.Lang
	if lang != "en" && lang != "de" && lang != "fa" {
		lang = "fa"
	}
	ch, err := s.issueOTP(r.Context(), models.AuthChallenge{
		Purpose:       "login",
		UserID:        u.ID,
		Username:      u.Username,
		TelegramPhone: u.TelegramPhone,
		Lang:          lang,
		ChatID:        u.TelegramChatID,
		Status:        "otp_sent",
		ExpiresAt:     time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.viewChallenge(ch))
}

func (s *Server) issueOTP(ctx context.Context, seed models.AuthChallenge) (models.AuthChallenge, error) {
	link, err := auth.RandomLink()
	if err != nil {
		return models.AuthChallenge{}, err
	}
	otp, err := auth.RandomOTP()
	if err != nil {
		return models.AuthChallenge{}, err
	}
	exp := time.Now().Add(5 * time.Minute)
	seed.LinkToken = link
	seed.OTPHash = auth.HashOTP("pending", otp)
	seed.OTPExpiresAt = &exp
	now := time.Now()
	seed.OTPSentAt = &now
	seed.Status = "otp_sent"
	ch, err := s.store.CreateChallenge(ctx, seed)
	if err != nil {
		return models.AuthChallenge{}, err
	}
	hash := auth.HashOTP(ch.ID, otp)
	if err := s.store.SetChallengeOTP(ctx, ch.ID, ch.ChatID, ch.PhoneFromTG, hash, exp); err != nil {
		return models.AuthChallenge{}, err
	}
	ch.OTPHash = hash
	ch.Status = "otp_sent"
	if err := s.bot.SendTo(ctx, ch.ChatID, otpMessage(ch.Lang, otp, ch.Username)); err != nil {
		return models.AuthChallenge{}, err
	}
	return ch, nil
}

func otpMessage(lang, otp, username string) string {
	switch lang {
	case "de":
		return "PVMoney Anmeldecode für " + username + ":\n\n" + otp + "\n\nGültig 5 Minuten. Wenn du das nicht warst, ignoriere diese Nachricht."
	case "en":
		return "PVMoney sign-in code for " + username + ":\n\n" + otp + "\n\nValid for 5 minutes. If this wasn’t you, ignore it."
	default:
		return "کد ورود پی‌وی‌مانی برای " + username + ":\n\n" + otp + "\n\nتا ۵ دقیقه معتبر است. اگر این کار را تو نکردی، نادیده بگیر."
	}
}

func (s *Server) verifyCode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"code"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	code := strings.TrimSpace(auth.FoldDigits(in.Code))
	if len(code) != 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_code"})
		return
	}
	ch, err := s.store.ChallengeByID(r.Context(), in.ChallengeID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if ch.Status == "consumed" || time.Now().After(ch.ExpiresAt) {
		writeJSON(w, http.StatusGone, map[string]string{"error": "expired"})
		return
	}
	if ch.Status != "otp_sent" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wait_telegram"})
		return
	}
	if ch.OTPExpiresAt != nil && time.Now().After(*ch.OTPExpiresAt) {
		writeJSON(w, http.StatusGone, map[string]string{"error": "expired"})
		return
	}
	if ch.Attempts >= 5 {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too_many_attempts"})
		return
	}
	if auth.HashOTP(ch.ID, code) != ch.OTPHash {
		n, _ := s.store.BumpChallengeAttempts(r.Context(), ch.ID)
		if n >= 5 {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too_many_attempts"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_code"})
		return
	}

	var u models.User
	switch ch.Purpose {
	case "register":
		u, err = s.store.CreateUser(r.Context(), ch.Username, ch.TelegramPhone, ch.ChatID)
	case "login", "reconnect":
		if ch.UserID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expired"})
			return
		}
		if ch.Purpose == "reconnect" && ch.ChatID != "" {
			_ = s.store.BindTelegram(r.Context(), ch.UserID, ch.ChatID, ch.TelegramPhone)
		}
		u, err = s.store.UserByID(r.Context(), ch.UserID)
		if err == nil {
			_ = s.store.TouchLogin(r.Context(), u.ID)
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expired"})
		return
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	_ = s.store.ConsumeChallenge(r.Context(), ch.ID)
	token, err := s.issueSession(r.Context(), u.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	auth.SetSessionCookie(w, r, token)
	if s.bot != nil {
		go func() {
			_ = s.bot.Welcome(context.Background())
		}()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  publicUser(u),
	})
}

func (s *Server) issueSession(ctx context.Context, userID string) (string, error) {
	tok, err := auth.RandomToken()
	if err != nil {
		return "", err
	}
	if err := s.store.CreateSession(ctx, userID, auth.Hash(tok), 30*24*time.Hour); err != nil {
		return "", err
	}
	return tok, nil
}

func (s *Server) resendCode(w http.ResponseWriter, r *http.Request) {
	if s.bot == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "telegram_disabled"})
		return
	}
	var in struct {
		ChallengeID string `json:"challenge_id"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	ch, err := s.store.ChallengeByID(r.Context(), in.ChallengeID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if ch.Status == "consumed" || time.Now().After(ch.ExpiresAt) {
		writeJSON(w, http.StatusGone, map[string]string{"error": "expired"})
		return
	}
	if ch.ChatID == "" || ch.Status != "otp_sent" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wait_telegram"})
		return
	}
	if ch.OTPSentAt != nil && time.Since(*ch.OTPSentAt) < 30*time.Second {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too_many_attempts"})
		return
	}
	otp, err := auth.RandomOTP()
	if err != nil {
		writeErr(w, err)
		return
	}
	exp := time.Now().Add(5 * time.Minute)
	if err := s.store.SetChallengeOTP(r.Context(), ch.ID, ch.ChatID, ch.PhoneFromTG, auth.HashOTP(ch.ID, otp), exp); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.bot.SendTo(r.Context(), ch.ChatID, otpMessage(ch.Lang, otp, ch.Username)); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "telegram_disabled"})
		return
	}
	ch.Status = "otp_sent"
	writeJSON(w, http.StatusOK, s.viewChallenge(ch))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if tok := auth.Bearer(r); tok != "" {
		_ = s.store.DeleteSession(r.Context(), auth.Hash(tok))
	}
	auth.ClearSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "logged_out"})
}
