package store

import (
	"context"
	"database/sql"
	"errors"

	"pvmoney/internal/models"
)

func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key=$1`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO app_settings (key, value) VALUES ($1,$2)
ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, key, value)
	return err
}

func (s *Store) TelegramSettings(ctx context.Context) (models.Settings, error) {
	chat, err := s.GetSetting(ctx, "telegram_chat_id")
	if err != nil {
		return models.Settings{}, err
	}
	user, err := s.GetSetting(ctx, "telegram_bot_username")
	if err != nil {
		return models.Settings{}, err
	}
	return models.Settings{
		TelegramChatID:    chat,
		TelegramConnected: chat != "",
		BotUsername:       user,
	}, nil
}
