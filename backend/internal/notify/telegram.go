package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"pvmoney/internal/store"
)

type Bot struct {
	token  string
	store  *store.Store
	client *http.Client
	offset int64
	mu     sync.Mutex
	sess   map[int64]*session
}

type session struct {
	Step      string
	Kind      string
	AccountID string
	ToID      string
	Category  string
	Amount    int64
	Note      string
	DebtID    string
	InstID    string
	Opts      []string
	MsgID     int
}

func Start(ctx context.Context, st *store.Store, token string) *Bot {
	if strings.TrimSpace(token) == "" {
		log.Println("telegram: no TELEGRAM_BOT_TOKEN, reminders disabled")
		return nil
	}
	b := &Bot{
		token:  strings.TrimSpace(token),
		store:  st,
		client: &http.Client{Timeout: 35 * time.Second},
		sess:   map[int64]*session{},
	}
	if u, err := b.getMe(); err == nil {
		_ = st.SetSetting(ctx, "telegram_bot_username", u)
		log.Printf("telegram: bot @%s ready", u)
	} else {
		log.Printf("telegram: getMe failed: %v", err)
	}
	_ = b.setCommands()
	go b.poll(ctx)
	go b.remindLoop(ctx)
	return b
}

func (b *Bot) poll(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		updates, err := b.getUpdates()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		for _, u := range updates {
			if u.UpdateID >= b.offset {
				b.offset = u.UpdateID + 1
			}
			b.handleUpdate(ctx, u)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, u tgUpdate) {
	if u.Callback != nil && u.Callback.Message != nil {
		chat := u.Callback.Message.Chat.ID
		if !b.allow(ctx, chat) {
			_ = b.answer(u.Callback.ID, "")
			return
		}
		_ = b.answer(u.Callback.ID, "")
		b.onCallback(ctx, chat, u.Callback.Message.MessageID, u.Callback.Data)
		return
	}
	if u.Message == nil {
		return
	}
	chat := u.Message.Chat.ID
	if u.Message.Contact != nil {
		b.handleContact(ctx, chat, u.Message)
		return
	}
	text := strings.TrimSpace(u.Message.Text)
	if b.handleAuthStart(ctx, chat, text) {
		return
	}
	if !b.allow(ctx, chat) {
		_ = b.sendTo(chat, tr("fa", "auth.needApp"), nil, false)
		return
	}
	b.onMessage(ctx, chat, text)
}

func (b *Bot) allow(ctx context.Context, chat int64) bool {
	id := strconv.FormatInt(chat, 10)
	if u, err := b.store.UserByChatID(ctx, id); err == nil && u.TelegramVerified {
		return true
	}
	saved, _ := b.store.GetSetting(ctx, "telegram_chat_id")
	return saved != "" && saved == id
}

func (b *Bot) get(chat int64) *session {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sess[chat]
	if s == nil {
		s = &session{}
		b.sess[chat] = s
	}
	return s
}

func (b *Bot) clear(chat int64) {
	b.mu.Lock()
	b.sess[chat] = &session{}
	b.mu.Unlock()
}

func (b *Bot) lang() string {
	v, _ := b.store.GetSetting(context.Background(), "telegram_ui_lang")
	if v == "en" || v == "de" || v == "fa" {
		return v
	}
	return "fa"
}

func (b *Bot) t(key string) string {
	return tr(b.lang(), key)
}

type tgResp struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

type tgUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

type tgContact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	UserID      int64  `json:"user_id"`
}

type tgMessage struct {
	MessageID int        `json:"message_id"`
	Text      string     `json:"text"`
	Chat      tgChat     `json:"chat"`
	From      *tgUser    `json:"from"`
	Contact   *tgContact `json:"contact"`
}

type tgCallback struct {
	ID      string     `json:"id"`
	Data    string     `json:"data"`
	Message *tgMessage `json:"message"`
}

type tgUpdate struct {
	UpdateID int64       `json:"update_id"`
	Message  *tgMessage  `json:"message"`
	Callback *tgCallback `json:"callback_query"`
}

type btn struct {
	Text string `json:"text"`
	Data string `json:"callback_data,omitempty"`
}

func (b *Bot) getMe() (string, error) {
	var u tgUser
	if err := b.call("getMe", nil, &u); err != nil {
		return "", err
	}
	return u.Username, nil
}

func (b *Bot) getUpdates() ([]tgUpdate, error) {
	body := map[string]any{
		"timeout":         25,
		"offset":          b.offset,
		"allowed_updates": []string{"message", "callback_query"},
	}
	var updates []tgUpdate
	if err := b.call("getUpdates", body, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func (b *Bot) setCommands() error {
	return b.call("setMyCommands", map[string]any{
		"commands": []map[string]string{
			{"command": "start", "description": "Open PVMoney menu"},
			{"command": "menu", "description": "Main menu"},
			{"command": "expense", "description": "Add expense"},
			{"command": "income", "description": "Add income"},
			{"command": "cancel", "description": "Cancel current step"},
		},
	}, nil)
}

func (b *Bot) destChats(ctx context.Context) []int64 {
	ids, err := b.store.VerifiedChatIDs(ctx)
	out := make([]int64, 0, 2)
	seen := map[int64]bool{}
	add := func(raw string) {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id == 0 || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	if err == nil {
		for _, id := range ids {
			add(id)
		}
	}
	if saved, e := b.store.GetSetting(ctx, "telegram_chat_id"); e == nil {
		add(saved)
	}
	return out
}

func (b *Bot) Send(ctx context.Context, text string) error {
	chats := b.destChats(ctx)
	if len(chats) == 0 {
		return fmt.Errorf("no chat id")
	}
	var last error
	ok := false
	for _, id := range chats {
		if err := b.sendTo(id, text, nil, false); err != nil {
			last = err
			continue
		}
		ok = true
	}
	if ok {
		return nil
	}
	return last
}

func (b *Bot) SendTo(_ context.Context, chatID, text string) error {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil || id == 0 {
		return fmt.Errorf("no chat id")
	}
	return b.sendTo(id, text, nil, false)
}

func (b *Bot) Welcome(ctx context.Context) error {
	chats := b.destChats(ctx)
	if len(chats) == 0 {
		return fmt.Errorf("no chat id")
	}
	var last error
	for _, id := range chats {
		if err := b.sendHome(ctx, id, 0, true); err != nil {
			last = err
		}
	}
	return last
}

func (b *Bot) sendMarkup(chat int64, text string, markup any) error {
	payload := map[string]any{
		"chat_id":                  chat,
		"text":                     text,
		"disable_web_page_preview": true,
	}
	if markup != nil {
		payload["reply_markup"] = markup
	}
	return b.call("sendMessage", payload, nil)
}

func (b *Bot) sendTo(chat int64, text string, inline [][]btn, withKB bool) error {
	var markup any
	if inline != nil {
		markup = inlineMarkup(inline)
	} else if withKB {
		markup = b.replyKeyboard()
	}
	return b.sendMarkup(chat, text, markup)
}

func (b *Bot) show(chat int64, msgID int, text string, inline [][]btn) {
	if msgID > 0 {
		payload := map[string]any{
			"chat_id":    chat,
			"message_id": msgID,
			"text":       text,
			"disable_web_page_preview": true,
		}
		if inline != nil {
			payload["reply_markup"] = inlineMarkup(inline)
		}
		if err := b.call("editMessageText", payload, nil); err == nil {
			return
		}
	}
	_ = b.sendTo(chat, text, inline, false)
}

func (b *Bot) answer(id, text string) error {
	body := map[string]any{"callback_query_id": id}
	if text != "" {
		body["text"] = text
	}
	return b.call("answerCallbackQuery", body, nil)
}

func inlineMarkup(rows [][]btn) map[string]any {
	out := make([][]map[string]string, 0, len(rows))
	for _, row := range rows {
		line := make([]map[string]string, 0, len(row))
		for _, b := range row {
			line = append(line, map[string]string{"text": b.Text, "callback_data": b.Data})
		}
		out = append(out, line)
	}
	return map[string]any{"inline_keyboard": out}
}

func (b *Bot) replyKeyboard() map[string]any {
	row := func(keys ...string) []map[string]string {
		out := make([]map[string]string, 0, len(keys))
		for _, k := range keys {
			out = append(out, map[string]string{"text": b.t(k)})
		}
		return out
	}
	return map[string]any{
		"keyboard": [][]map[string]string{
			row("kb.overview"),
			row("kb.accounts", "kb.assets"),
			row("kb.debts", "kb.projects"),
			row("kb.expense", "kb.income"),
			row("kb.transfer", "kb.menu"),
		},
		"resize_keyboard": true,
		"is_persistent":   true,
	}
}

func (b *Bot) call(method string, payload any, dest any) error {
	var rdr io.Reader
	if payload != nil {
		raw, _ := json.Marshal(payload)
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+b.token+"/"+method, rdr)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var wrap tgResp
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return err
	}
	if !wrap.OK {
		return fmt.Errorf("telegram %s failed: %s", method, string(raw))
	}
	if dest != nil && len(wrap.Result) > 0 {
		return json.Unmarshal(wrap.Result, dest)
	}
	return nil
}

func (b *Bot) remindLoop(ctx context.Context) {
	b.runReminders(ctx)
	t := time.NewTicker(15 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.runReminders(ctx)
		}
	}
}

func (b *Bot) runReminders(ctx context.Context) {
	chat, err := b.store.GetSetting(ctx, "telegram_chat_id")
	if err != nil || chat == "" {
		return
	}
	items, err := b.store.PendingReminders(ctx)
	if err != nil {
		log.Printf("telegram reminders: %v", err)
		return
	}
	loc, _ := time.LoadLocation("Asia/Tehran")
	if loc == nil {
		loc = time.FixedZone("IRST", int(3.5*3600))
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	for _, it := range items {
		due := it.DueDate.In(loc)
		dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, loc)
		days := int(dueDay.Sub(today).Hours() / 24)
		kind := ""
		switch days {
		case 20:
			kind = "d20"
		case 7:
			kind = "d7"
		case 3:
			kind = "d3"
		case 0:
			if now.Hour() < 7 {
				continue
			}
			kind = "due"
		default:
			continue
		}
		sent, err := b.store.ReminderSent(ctx, it.ID, kind)
		if err != nil || sent {
			continue
		}
		if err := b.Send(ctx, formatReminder(it, kind, loc)); err != nil {
			log.Printf("telegram send: %v", err)
			continue
		}
		_ = b.store.MarkReminder(ctx, it.ID, kind)
	}
}
