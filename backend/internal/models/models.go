package models

import "time"

// MaxMoney is 10 trillion Toman — keeps JSON/int64 and UI math in a sane range.
const MaxMoney int64 = 10_000_000_000_000

type Account struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	BankName      string    `json:"bank_name"`
	AccountNumber string    `json:"account_number"`
	Type          string    `json:"type"`
	Balance       int64     `json:"balance"`
	Color         string    `json:"color"`
	Icon          string    `json:"icon"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ProjectItem struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	Name          string    `json:"name"`
	PlannedAmount int64     `json:"planned_amount"`
	PaidAmount    int64     `json:"paid_amount"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Asset struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Quantity        float64   `json:"quantity"`
	Unit            string    `json:"unit"`
	UnitValue       int64     `json:"unit_value"`
	Value           int64     `json:"value"`
	Notes           string    `json:"notes"`
	Color           string    `json:"color"`
	SourceProjectID *string   `json:"source_project_id,omitempty"`
	SourceProject   string    `json:"source_project_name,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Project struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	TargetAmount    int64         `json:"target_amount"`
	CurrentAmount   int64         `json:"current_amount"`
	BaseAmount      int64         `json:"base_amount"`
	Deadline        *time.Time    `json:"deadline,omitempty"`
	Color           string        `json:"color"`
	Status          string        `json:"status"`
	AssetID         *string       `json:"asset_id,omitempty"`
	ResultAssetType string        `json:"result_asset_type"`
	Items           []ProjectItem `json:"items,omitempty"`
	Asset           *Asset        `json:"asset,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type Transaction struct {
	ID          string    `json:"id"`
	AccountID   *string   `json:"account_id,omitempty"`
	ToAccountID *string   `json:"to_account_id,omitempty"`
	ProjectID   *string   `json:"project_id,omitempty"`
	ItemID      *string   `json:"item_id,omitempty"`
	Type        string    `json:"type"`
	Amount      int64     `json:"amount"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
	CreatedAt   time.Time `json:"created_at"`
	AccountName string    `json:"account_name,omitempty"`
	ToName      string    `json:"to_account_name,omitempty"`
	ProjectName string    `json:"project_name,omitempty"`
}

type DailyPoint struct {
	Date    string `json:"date"`
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
}

type CategoryPoint struct {
	Category string `json:"category"`
	Amount   int64  `json:"amount"`
}

type MonthlyPoint struct {
	Month   string `json:"month"`
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
}

type Dashboard struct {
	NetWorth       int64           `json:"net_worth"`
	Liquid         int64           `json:"liquid"`
	AssetsTotal    int64           `json:"assets_total"`
	ProjectSpend   int64           `json:"project_spend"`
	MonthlyIncome  int64           `json:"monthly_income"`
	MonthlyExpense int64           `json:"monthly_expense"`
	AccountCount   int             `json:"account_count"`
	ProjectCount   int             `json:"project_count"`
	Accounts       []Account       `json:"accounts"`
	Projects       []Project       `json:"projects"`
	Assets         []Asset         `json:"assets"`
	DebtsRemaining int64           `json:"debts_remaining"`
	Upcoming       []Installment   `json:"upcoming"`
	Recent         []Transaction   `json:"recent"`
	Cashflow       []DailyPoint    `json:"cashflow"`
	CategorySpend  []CategoryPoint `json:"category_spend"`
	Monthly        []MonthlyPoint  `json:"monthly"`
}

type AccountInput struct {
	Name          string `json:"name"`
	BankName      string `json:"bank_name"`
	AccountNumber string `json:"account_number"`
	Type          string `json:"type"`
	Balance       int64  `json:"balance"`
	Color         string `json:"color"`
	Icon          string `json:"icon"`
}

type ProjectInput struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	TargetAmount    int64   `json:"target_amount"`
	BaseAmount      int64   `json:"base_amount"`
	Deadline        *string `json:"deadline"`
	Color           string  `json:"color"`
	ResultAssetType string  `json:"result_asset_type"`
}

type ItemInput struct {
	Name          string `json:"name"`
	PlannedAmount int64  `json:"planned_amount"`
	Notes         string `json:"notes"`
}

type PayInput struct {
	AccountID   string `json:"account_id"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

type AssetInput struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Quantity        float64 `json:"quantity"`
	Unit            string  `json:"unit"`
	UnitValue       int64   `json:"unit_value"`
	Notes           string  `json:"notes"`
	Color           string  `json:"color"`
	SourceProjectID *string `json:"source_project_id"`
}

type ConvertAssetInput struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	UnitValue int64   `json:"unit_value"`
	Notes     string  `json:"notes"`
	Color     string  `json:"color"`
}

type TransactionInput struct {
	AccountID   *string `json:"account_id"`
	ToAccountID *string `json:"to_account_id"`
	ProjectID   *string `json:"project_id"`
	ItemID      *string `json:"item_id"`
	Type        string  `json:"type"`
	Amount      int64   `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	OccurredAt  *string `json:"occurred_at"`
}

type ContributeInput struct {
	AccountID   string `json:"account_id"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

type AdjustInput struct {
	Balance     int64  `json:"balance"`
	Description string `json:"description"`
}

type Debt struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Type          string        `json:"type"`
	Creditor      string        `json:"creditor"`
	TotalAmount   int64         `json:"total_amount"`
	Remaining     int64         `json:"remaining"`
	Notes         string        `json:"notes"`
	Color         string        `json:"color"`
	HasSchedule   bool          `json:"has_schedule"`
	StartDate     *time.Time    `json:"start_date,omitempty"`
	EndDate       *time.Time    `json:"end_date,omitempty"`
	MonthlyAmount int64         `json:"monthly_amount"`
	DueDay        int           `json:"due_day"`
	Status        string        `json:"status"`
	NextDue       *time.Time    `json:"next_due,omitempty"`
	Installments  []Installment `json:"installments,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type Installment struct {
	ID        string     `json:"id"`
	DebtID    string     `json:"debt_id"`
	DebtName  string     `json:"debt_name,omitempty"`
	Creditor  string     `json:"creditor,omitempty"`
	Amount    int64      `json:"amount"`
	DueDate   time.Time  `json:"due_date"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`
	Status    string     `json:"status"`
	AccountID *string    `json:"account_id,omitempty"`
}

type DebtInput struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Creditor      string  `json:"creditor"`
	TotalAmount   int64   `json:"total_amount"`
	Notes         string  `json:"notes"`
	Color         string  `json:"color"`
	HasSchedule   bool    `json:"has_schedule"`
	StartDate     *string `json:"start_date"`
	EndDate       *string `json:"end_date"`
	MonthlyAmount int64   `json:"monthly_amount"`
	DueDay        int     `json:"due_day"`
}

type Settings struct {
	TelegramChatID   string `json:"telegram_chat_id"`
	TelegramConnected bool  `json:"telegram_connected"`
	BotUsername      string `json:"bot_username"`
}
