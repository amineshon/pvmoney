package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Rates struct {
	USDToman    int64  `json:"usd_toman"`
	EURToman    int64  `json:"eur_toman"`
	Gold18Toman int64  `json:"gold18_toman"`
	Gold24Toman int64  `json:"gold24_toman"`
	Source      string `json:"source"`
	UpdatedAt   string `json:"updated_at"`
}

type snapshot struct {
	Current map[string]struct {
		P json.RawMessage `json:"p"`
	} `json:"current"`
}

var (
	mu      sync.Mutex
	cached  *Rates
	cachedAt time.Time
)

const cacheTTL = 5 * time.Minute

func AssetValue(typ string, qty float64, book int64, r *Rates) int64 {
	if r == nil {
		return book
	}
	if qty <= 0 {
		qty = 1
	}
	var unit int64
	switch typ {
	case "gold", "gold18":
		unit = r.Gold18Toman
	case "gold24":
		unit = r.Gold24Toman
	case "usd":
		unit = r.USDToman
	case "eur":
		unit = r.EURToman
	}
	if unit <= 0 {
		return book
	}
	return int64(math.Round(qty * float64(unit)))
}

func Get(ctx context.Context) (Rates, error) {
	mu.Lock()
	defer mu.Unlock()
	if cached != nil && time.Since(cachedAt) < cacheTTL {
		return *cached, nil
	}
	out, err := fetch(ctx)
	if err != nil {
		if cached != nil {
			return *cached, nil
		}
		return Rates{}, err
	}
	cached = &out
	cachedAt = time.Now()
	return out, nil
}

func fetch(ctx context.Context) (Rates, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://call5.tgju.org/ajax.json", nil)
	if err != nil {
		return Rates{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 PVMoney/1.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return Rates{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return Rates{}, fmt.Errorf("tgju status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return Rates{}, err
	}
	var snap snapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		return Rates{}, err
	}
	gold18, err := rialToToman(snap.price("geram18"))
	if err != nil {
		return Rates{}, fmt.Errorf("gold18: %w", err)
	}
	gold24, err := rialToToman(snap.price("geram24"))
	if err != nil {
		return Rates{}, fmt.Errorf("gold24: %w", err)
	}
	usd, err := rialToToman(snap.price("price_dollar_rl"))
	if err != nil {
		return Rates{}, fmt.Errorf("usd: %w", err)
	}
	eur, err := rialToToman(snap.price("price_eur"))
	if err != nil {
		return Rates{}, fmt.Errorf("eur: %w", err)
	}
	return Rates{
		USDToman:    usd,
		EURToman:    eur,
		Gold18Toman: gold18,
		Gold24Toman: gold24,
		Source:      "tgju",
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s snapshot) price(key string) json.RawMessage {
	if s.Current == nil {
		return nil
	}
	return s.Current[key].P
}

func rialToToman(raw json.RawMessage) (int64, error) {
	n, err := parseCommaInt(raw)
	if err != nil {
		return 0, err
	}
	return n / 10, nil
}

func parseCommaInt(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("empty price")
	}
	s := strings.TrimSpace(string(raw))
	s = strings.Trim(s, `"`)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "٬", "")
	if s == "" || s == "null" {
		return 0, fmt.Errorf("empty price")
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		f, ferr := strconv.ParseFloat(s, 64)
		if ferr != nil {
			return 0, err
		}
		n = int64(f)
	}
	if n <= 0 {
		return 0, fmt.Errorf("non-positive price")
	}
	return n, nil
}
