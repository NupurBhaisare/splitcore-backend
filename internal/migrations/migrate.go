package migrations

import (
	"log"

	"github.com/nupurbhaisare/splitcore-backend/internal/database"
)

func RunAll() error {
	log.Println("Running migrations...")

	if err := runMigration001(); err != nil {
		return err
	}

	if err := runMigration002(); err != nil {
		return err
	}

	if err := runMigration003(); err != nil {
		return err
	}

	log.Println("All migrations completed successfully")
	return nil
}

func runMigration003() error {
	// Migration 3 — Multi-Currency, Comments, Summaries
	migrations := []string{
		// Currencies table
		`CREATE TABLE IF NOT EXISTS currencies (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			symbol TEXT NOT NULL,
			decimal_places INTEGER NOT NULL DEFAULT 2,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		// Exchange rates table
		`CREATE TABLE IF NOT EXISTS exchange_rates (
			id TEXT PRIMARY KEY,
			from_currency TEXT NOT NULL,
			to_currency TEXT NOT NULL,
			rate REAL NOT NULL,
			fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(from_currency, to_currency)
		)`,

		// Expense comments table
		`CREATE TABLE IF NOT EXISTS expense_comments (
			id TEXT PRIMARY KEY,
			expense_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			body TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL,
			FOREIGN KEY (expense_id) REFERENCES expenses(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,

		// Monthly summaries table
		`CREATE TABLE IF NOT EXISTS monthly_summaries (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			group_id TEXT NOT NULL,
			year INTEGER NOT NULL,
			month INTEGER NOT NULL,
			total_paid_cents INTEGER NOT NULL DEFAULT 0,
			total_owed_cents INTEGER NOT NULL DEFAULT 0,
			total_settled_cents INTEGER NOT NULL DEFAULT 0,
			expense_count INTEGER NOT NULL DEFAULT 0,
			currency_code TEXT NOT NULL DEFAULT 'USD',
			UNIQUE(user_id, group_id, year, month)
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_currencies_code ON currencies(code)`,
		`CREATE INDEX IF NOT EXISTS idx_exchange_rates_currencies ON exchange_rates(from_currency, to_currency)`,
		`CREATE INDEX IF NOT EXISTS idx_expense_comments_expense ON expense_comments(expense_id)`,
		`CREATE INDEX IF NOT EXISTS idx_expense_comments_user ON expense_comments(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_expense_comments_deleted ON expense_comments(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_monthly_summaries_user ON monthly_summaries(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_monthly_summaries_group ON monthly_summaries(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_monthly_summaries_year_month ON monthly_summaries(year, month)`,
		`CREATE INDEX IF NOT EXISTS idx_monthly_summaries_lookup ON monthly_summaries(user_id, group_id, year, month)`,

		// Add rich split columns to existing expense_splits table
		`ALTER TABLE expense_splits ADD COLUMN percentage REAL DEFAULT 0`,
		`ALTER TABLE expense_splits ADD COLUMN share_count INTEGER DEFAULT 0`,
	}

	for _, m := range migrations {
		if _, err := database.DB.Exec(m); err != nil {
			return err
		}
	}

	// Seed common currencies
	currencies := []struct {
		code         string
		name         string
		symbol       string
		decimalPlaces int
	}{
		{"USD", "US Dollar", "$", 2},
		{"EUR", "Euro", "€", 2},
		{"GBP", "British Pound", "£", 2},
		{"JPY", "Japanese Yen", "¥", 0},
		{"CAD", "Canadian Dollar", "C$", 2},
		{"AUD", "Australian Dollar", "A$", 2},
		{"CHF", "Swiss Franc", "CHF", 2},
		{"CNY", "Chinese Yuan", "¥", 2},
		{"INR", "Indian Rupee", "₹", 2},
		{"MXN", "Mexican Peso", "MX$", 2},
		{"BRL", "Brazilian Real", "R$", 2},
		{"KRW", "South Korean Won", "₩", 0},
		{"SGD", "Singapore Dollar", "S$", 2},
		{"HKD", "Hong Kong Dollar", "HK$", 2},
		{"NOK", "Norwegian Krone", "kr", 2},
		{"SEK", "Swedish Krona", "kr", 2},
		{"DKK", "Danish Krone", "kr", 2},
		{"NZD", "New Zealand Dollar", "NZ$", 2},
		{"ZAR", "South African Rand", "R", 2},
		{"THB", "Thai Baht", "฿", 2},
	}

	for _, c := range currencies {
		database.DB.Exec(
			`INSERT OR IGNORE INTO currencies (code, name, symbol, decimal_places, updated_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			c.code, c.name, c.symbol, c.decimalPlaces,
		)
	}

	// Seed default exchange rates (USD as base)
	defaultRates := []struct {
		from string
		to   string
		rate float64
	}{
		{"USD", "USD", 1.0},
		{"USD", "EUR", 0.92},
		{"USD", "GBP", 0.79},
		{"USD", "JPY", 149.50},
		{"USD", "CAD", 1.36},
		{"USD", "AUD", 1.53},
		{"USD", "CHF", 0.88},
		{"USD", "CNY", 7.24},
		{"USD", "INR", 83.12},
		{"USD", "MXN", 17.15},
		{"USD", "BRL", 4.97},
		{"USD", "KRW", 1328.50},
		{"USD", "SGD", 1.34},
		{"USD", "HKD", 7.82},
		{"USD", "NOK", 10.72},
		{"USD", "SEK", 10.45},
		{"USD", "DKK", 6.87},
		{"USD", "NZD", 1.64},
		{"USD", "ZAR", 18.65},
		{"USD", "THB", 35.50},
		{"EUR", "USD", 1.087},
		{"GBP", "USD", 1.266},
	}

	for _, r := range defaultRates {
		database.DB.Exec(
			`INSERT OR REPLACE INTO exchange_rates (id, from_currency, to_currency, rate, fetched_at)
			 VALUES (lower(hex(randomblob(8))), ?, ?, ?, CURRENT_TIMESTAMP)`,
			r.from, r.to, r.rate,
		)
	}

	log.Println("Migration 003 (Multi-Currency, Comments, Summaries) applied")
	return nil
}

func runMigration001() error {
	// Migration 1 — Foundation Schema
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			icon_emoji TEXT NOT NULL DEFAULT '💰',
			currency_code TEXT NOT NULL DEFAULT 'USD',
			created_by_user_id TEXT NOT NULL,
			invite_code TEXT UNIQUE NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL,
			FOREIGN KEY (created_by_user_id) REFERENCES users(id)
		)`,

		`CREATE TABLE IF NOT EXISTS group_members (
			id TEXT PRIMARY KEY,
			group_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			nickname_in_group TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'member',
			joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			removed_at DATETIME DEFAULT NULL,
			FOREIGN KEY (group_id) REFERENCES groups(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(group_id, user_id)
		)`,

		`CREATE TABLE IF NOT EXISTS expenses (
			id TEXT PRIMARY KEY,
			group_id TEXT NOT NULL,
			paid_by_user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			amount_cents INTEGER NOT NULL,
			currency_code TEXT NOT NULL DEFAULT 'USD',
			category TEXT NOT NULL DEFAULT 'other',
			expense_date DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL,
			FOREIGN KEY (group_id) REFERENCES groups(id),
			FOREIGN KEY (paid_by_user_id) REFERENCES users(id)
		)`,

		`CREATE TABLE IF NOT EXISTS expense_splits (
			id TEXT PRIMARY KEY,
			expense_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			share_amount_cents INTEGER NOT NULL,
			split_type TEXT NOT NULL DEFAULT 'equal',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (expense_id) REFERENCES expenses(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(expense_id, user_id)
		)`,

		// Indexes for common queries
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_deleted ON users(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_groups_created_by ON groups(created_by_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_groups_deleted ON groups(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_group_members_group ON group_members(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_group_members_user ON group_members(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_expenses_group ON expenses(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_expenses_paid_by ON expenses(paid_by_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_expenses_deleted ON expenses(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_expense_splits_expense ON expense_splits(expense_id)`,
		`CREATE INDEX IF NOT EXISTS idx_expense_splits_user ON expense_splits(user_id)`,
	}

	for _, m := range migrations {
		if _, err := database.DB.Exec(m); err != nil {
			return err
		}
	}

	log.Println("Migration 001 (Foundation Schema) applied")
	return nil
}

func runMigration002() error {
	// Migration 2 — Settlements, Activities, Push Tokens
	migrations := []string{
		// Settlements table
		`CREATE TABLE IF NOT EXISTS settlements (
			id TEXT PRIMARY KEY,
			group_id TEXT NOT NULL,
			from_user_id TEXT NOT NULL,
			to_user_id TEXT NOT NULL,
			amount_cents INTEGER NOT NULL,
			currency_code TEXT NOT NULL DEFAULT 'USD',
			settled_at DATETIME NOT NULL,
			created_by_user_id TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			payment_method TEXT NOT NULL DEFAULT 'cash',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES groups(id),
			FOREIGN KEY (from_user_id) REFERENCES users(id),
			FOREIGN KEY (to_user_id) REFERENCES users(id),
			FOREIGN KEY (created_by_user_id) REFERENCES users(id)
		)`,

		// Activities table
		`CREATE TABLE IF NOT EXISTS activities (
			id TEXT PRIMARY KEY,
			group_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			target_user_ids TEXT NOT NULL DEFAULT '[]',
			activity_type TEXT NOT NULL,
			metadata TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES groups(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,

		// Push tokens table
		`CREATE TABLE IF NOT EXISTS push_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			device_token TEXT NOT NULL,
			platform TEXT NOT NULL DEFAULT 'ios',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(user_id, device_token)
		)`,

		// Notifications table
		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			data TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			read_at DATETIME DEFAULT NULL,
			sent_via TEXT NOT NULL DEFAULT 'apns',
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_settlements_group ON settlements(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_settlements_from_user ON settlements(from_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_settlements_to_user ON settlements(to_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_settlements_deleted ON settlements(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_activities_group ON activities(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_activities_user ON activities(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_activities_created ON activities(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_push_tokens_user ON push_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_push_tokens_deleted ON push_tokens(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read_at)`,
	}

	for _, m := range migrations {
		if _, err := database.DB.Exec(m); err != nil {
			return err
		}
	}

	log.Println("Migration 002 (Settlements, Activities, Push Tokens) applied")
	return nil
}
