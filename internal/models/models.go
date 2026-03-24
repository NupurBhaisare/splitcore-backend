package models

import (
	"time"
)

// User represents a user in the system.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	AvatarURL    string    `json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// Group represents a shared expense group.
type Group struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	IconEmoji     string    `json:"icon_emoji"`
	CurrencyCode  string    `json:"currency_code"`
	CreatedByUser string    `json:"created_by_user_id"`
	InviteCode    string    `json:"invite_code"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// GroupMember represents a user's membership in a group.
type GroupMember struct {
	ID        string     `json:"id"`
	GroupID   string     `json:"group_id"`
	UserID    string     `json:"user_id"`
	Nickname  string     `json:"nickname_in_group"`
	Role      string     `json:"role"` // owner, member
	JoinedAt  time.Time  `json:"joined_at"`
	RemovedAt *time.Time `json:"removed_at,omitempty"`

	// Joined fields for responses
	User *User `json:"user,omitempty"`
}

// Expense represents a shared expense in a group.
type Expense struct {
	ID            string    `json:"id"`
	GroupID       string    `json:"group_id"`
	PaidByUserID  string    `json:"paid_by_user_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	AmountCents   int64     `json:"amount_cents"`
	CurrencyCode  string    `json:"currency_code"`
	Category      string    `json:"category"` // food, travel, shopping, entertainment, other
	ExpenseDate   time.Time `json:"expense_date"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`

	// Joined fields for responses
	PaidBy *User          `json:"paid_by,omitempty"`
	Splits []ExpenseSplit `json:"splits,omitempty"`
}

// ExpenseSplit represents how an expense is split among users.
type ExpenseSplit struct {
	ID               string    `json:"id"`
	ExpenseID        string    `json:"expense_id"`
	UserID           string    `json:"user_id"`
	ShareAmountCents int64     `json:"share_amount_cents"`
	SplitType        string    `json:"split_type"` // equal, percentage, exact, shares
	Percentage       float64   `json:"percentage,omitempty"` // for percentage splits (0-100)
	ShareCount       int       `json:"share_count,omitempty"` // for shares splits
	CreatedAt        time.Time `json:"created_at"`

	// Joined fields
	User *User `json:"user,omitempty"`
}

// Currency represents a supported currency.
type Currency struct {
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	DecimalPlaces int       `json:"decimal_places"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ExchangeRate represents a currency exchange rate.
type ExchangeRate struct {
	ID           string    `json:"id"`
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	Rate         float64   `json:"rate"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// ExpenseComment represents a comment on an expense.
type ExpenseComment struct {
	ID        string     `json:"id"`
	ExpenseID string     `json:"expense_id"`
	UserID    string     `json:"user_id"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// Joined fields
	User *User `json:"user,omitempty"`
}

// MonthlySummary represents a user's monthly expense summary.
type MonthlySummary struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	GroupID         string `json:"group_id"`
	Year            int    `json:"year"`
	Month           int    `json:"month"`
	TotalPaidCents  int64  `json:"total_paid_cents"`
	TotalOwedCents  int64  `json:"total_owed_cents"`
	TotalSettledCents int64 `json:"total_settled_cents"`
	ExpenseCount    int    `json:"expense_count"`
	CurrencyCode    string `json:"currency_code"`
}

// SearchExpenseResult represents a search result for an expense.
type SearchExpenseResult struct {
	Expenses     []Expense   `json:"expenses"`
	Total        int         `json:"total"`
	Page         int         `json:"page"`
	PerPage      int         `json:"per_page"`
}

// Balance represents a user's net balance in a group.
type Balance struct {
	UserID       string `json:"user_id"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url"`
	NetCents     int64  `json:"net_cents"`
	Status       string `json:"status"` // owed, owes, settled
}

// Settlement represents a recorded settlement between two users in a group.
type Settlement struct {
	ID              string    `json:"id"`
	GroupID         string    `json:"group_id"`
	FromUserID      string    `json:"from_user_id"`
	ToUserID        string    `json:"to_user_id"`
	AmountCents     int64     `json:"amount_cents"`
	CurrencyCode    string    `json:"currency_code"`
	SettledAt       time.Time `json:"settled_at"`
	CreatedByUserID string    `json:"created_by_user_id"`
	Note            string    `json:"note"`
	PaymentMethod   string    `json:"payment_method"` // cash, app, other
	CreatedAt       time.Time `json:"created_at"`

	// Joined fields
	FromUser *User `json:"from_user,omitempty"`
	ToUser   *User `json:"to_user,omitempty"`
}

// Activity represents an activity feed item in a group.
type Activity struct {
	ID            string                 `json:"id"`
	GroupID       string                 `json:"group_id"`
	UserID        string                 `json:"user_id"` // actor who triggered the activity
	TargetUserIDs []string               `json:"target_user_ids,omitempty"`
	ActivityType  string                 `json:"activity_type"` // expense_added, expense_edited, expense_deleted, member_joined, member_left, settlement_made, group_created
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	ReadAt        *time.Time             `json:"read_at,omitempty"`

	// Joined fields
	User *User `json:"user,omitempty"`
}

// PushToken represents a device push token for a user.
type PushToken struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	DeviceToken string    `json:"device_token"`
	Platform    string    `json:"platform"` // ios, android
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// SimplifiedDebt represents a single debt transaction after netting.
type SimplifiedDebt struct {
	FromUserID  string `json:"from_user_id"`
	ToUserID    string `json:"to_user_id"`
	AmountCents int64  `json:"amount_cents"`
}

// DebtResponse represents the simplified debt response for a group.
type DebtResponse struct {
	GroupID      string           `json:"group_id"`
	CurrencyCode string          `json:"currency_code"`
	Balances     []Balance       `json:"balances"`
	Debts        []SimplifiedDebt `json:"debts"`
}

// Auth tokens
type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// API response wrappers
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type PaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Total   int         `json:"total,omitempty"`
	Page    int         `json:"page,omitempty"`
	PerPage int         `json:"per_page,omitempty"`
}
