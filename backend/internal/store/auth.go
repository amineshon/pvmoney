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

var challengeCols = `id, purpose, COALESCE(user_id::text,''), username, telegram_phone, lang, link_token, otp_hash, otp_expires_at, chat_id, phone_from_tg, status, attempts, otp_sent_at, expires_at, created_at`

func scanChallenge(row interface{ Scan(dest ...any) error }) (models.AuthChallenge, error) {
	var c models.AuthChallenge
	var otpExp, otpSent sql.NullTime
	err := row.Scan(&c.ID, &c.Purpose, &c.UserID, &c.Username, &c.TelegramPhone, &c.Lang, &c.LinkToken, &c.OTPHash, &otpExp, &c.ChatID, &c.PhoneFromTG, &c.Status, &c.Attempts, &otpSent, &c.ExpiresAt, &c.CreatedAt)
	if otpExp.Valid {
		c.OTPExpiresAt = &otpExp.Time
	}
	if otpSent.Valid {
		c.OTPSentAt = &otpSent.Time
	}
	return c, err
}

func scanUser(row interface{ Scan(dest ...any) error }) (models.User, error) {
	var u models.User
	var last sql.NullTime
	err := row.Scan(&u.ID, &u.Username, &u.TelegramPhone, &u.TelegramChatID, &u.TelegramVerified, &u.CreatedAt, &last)
	if last.Valid {
		u.LastLoginAt = &last.Time
	}
	return u, err
}

func (s *Store) HasUsers(ctx context.Context) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n > 0, err
}

func (s *Store) UserByID(ctx context.Context, id string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, username, telegram_phone, telegram_chat_id, telegram_verified, created_at, last_login_at
FROM users WHERE id=$1`, id)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByUsername(ctx context.Context, username string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, username, telegram_phone, telegram_chat_id, telegram_verified, created_at, last_login_at
FROM users WHERE LOWER(username)=LOWER($1)`, strings.TrimSpace(username))
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByPhone(ctx context.Context, phone string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, username, telegram_phone, telegram_chat_id, telegram_verified, created_at, last_login_at
FROM users WHERE telegram_phone=$1`, phone)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByChatID(ctx context.Context, chatID string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, username, telegram_phone, telegram_chat_id, telegram_verified, created_at, last_login_at
FROM users WHERE telegram_chat_id=$1`, chatID)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) UserBySession(ctx context.Context, tokenHash string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT u.id, u.username, u.telegram_phone, u.telegram_chat_id, u.telegram_verified, u.created_at, u.last_login_at
FROM sessions s JOIN users u ON u.id=s.user_id
WHERE s.token_hash=$1 AND s.expires_at > NOW()`, tokenHash)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrUnauthorized
	}
	return u, err
}

func (s *Store) VerifiedChatIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT telegram_chat_id FROM users
WHERE telegram_verified AND telegram_chat_id <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) CreateUser(ctx context.Context, username, phone, chatID string) (models.User, error) {
	now := time.Now()
	u := models.User{
		ID:               uuid.NewString(),
		Username:         username,
		TelegramPhone:    phone,
		TelegramChatID:   chatID,
		TelegramVerified: chatID != "",
		CreatedAt:        now,
		LastLoginAt:      &now,
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO users (id, username, telegram_phone, telegram_chat_id, telegram_verified, created_at, last_login_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)`, u.ID, u.Username, u.TelegramPhone, u.TelegramChatID, u.TelegramVerified, u.CreatedAt, now)
	if err != nil {
		if isUnique(err) {
			return models.User{}, fmt.Errorf("%w: username_taken", ErrConflict)
		}
		return models.User{}, err
	}
	if chatID != "" {
		_ = s.SetSetting(ctx, "telegram_chat_id", chatID)
	}
	return u, nil
}

func (s *Store) TouchLogin(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET last_login_at=NOW() WHERE id=$1`, userID)
	return err
}

func (s *Store) BindTelegram(ctx context.Context, userID, chatID, phone string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE users SET telegram_chat_id=$2, telegram_phone=CASE WHEN $3='' THEN telegram_phone ELSE $3 END, telegram_verified=TRUE
WHERE id=$1`, userID, chatID, phone)
	if err == nil && chatID != "" {
		_ = s.SetSetting(ctx, "telegram_chat_id", chatID)
	}
	return err
}

func (s *Store) CreateSession(ctx context.Context, userID, tokenHash string, ttl time.Duration) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO sessions (id, user_id, token_hash, expires_at) VALUES ($1,$2,$3,$4)`,
		uuid.NewString(), userID, tokenHash, time.Now().Add(ttl))
	return err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=$1`, tokenHash)
	return err
}

func (s *Store) CreateChallenge(ctx context.Context, c models.AuthChallenge) (models.AuthChallenge, error) {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO auth_challenges (
  id, purpose, user_id, username, telegram_phone, lang, link_token, otp_hash, otp_expires_at,
  chat_id, phone_from_tg, status, attempts, otp_sent_at, expires_at, created_at
) VALUES ($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		c.ID, c.Purpose, c.UserID, c.Username, c.TelegramPhone, c.Lang, c.LinkToken, c.OTPHash, nullTime(c.OTPExpiresAt),
		c.ChatID, c.PhoneFromTG, c.Status, c.Attempts, nullTime(c.OTPSentAt), c.ExpiresAt, c.CreatedAt)
	return c, err
}

func (s *Store) ChallengeByID(ctx context.Context, id string) (models.AuthChallenge, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+challengeCols+` FROM auth_challenges WHERE id=$1`, id)
	c, err := scanChallenge(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.AuthChallenge{}, ErrNotFound
	}
	return c, err
}

func (s *Store) ChallengeByLinkToken(ctx context.Context, token string) (models.AuthChallenge, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+challengeCols+` FROM auth_challenges WHERE link_token=$1 AND expires_at > NOW()`, token)
	c, err := scanChallenge(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.AuthChallenge{}, ErrNotFound
	}
	return c, err
}

func (s *Store) ChallengeByChatPending(ctx context.Context, chatID string) (models.AuthChallenge, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT `+challengeCols+` FROM auth_challenges
WHERE chat_id=$1 AND expires_at > NOW() AND status IN ('pending_start','waiting_contact','phone_mismatch','otp_sent')
ORDER BY created_at DESC LIMIT 1`, chatID)
	c, err := scanChallenge(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.AuthChallenge{}, ErrNotFound
	}
	return c, err
}

func (s *Store) SetChallengeStart(ctx context.Context, id, chatID string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE auth_challenges SET chat_id=$2, status='waiting_contact'
WHERE id=$1 AND expires_at > NOW()`, id, chatID)
	return err
}

func (s *Store) SetChallengeMismatch(ctx context.Context, id, phoneFromTG string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE auth_challenges SET phone_from_tg=$2, status='phone_mismatch'
WHERE id=$1`, id, phoneFromTG)
	return err
}

func (s *Store) SetChallengeOTP(ctx context.Context, id, chatID, phoneFromTG, otpHash string, otpExp time.Time) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
UPDATE auth_challenges
SET chat_id=$2, phone_from_tg=$3, otp_hash=$4, otp_expires_at=$5, otp_sent_at=$6, status='otp_sent', attempts=0
WHERE id=$1`, id, chatID, phoneFromTG, otpHash, otpExp, now)
	return err
}

func (s *Store) BumpChallengeAttempts(ctx context.Context, id string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
UPDATE auth_challenges SET attempts=attempts+1 WHERE id=$1 RETURNING attempts`, id).Scan(&n)
	return n, err
}

func (s *Store) ConsumeChallenge(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE auth_challenges SET status='consumed' WHERE id=$1`, id)
	return err
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func isUnique(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}
