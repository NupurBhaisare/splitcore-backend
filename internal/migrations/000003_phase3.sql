-- Migration 003: Multi-Currency, Comments, Monthly Summaries

-- Currencies table
CREATE TABLE IF NOT EXISTS currencies (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    decimal_places INTEGER NOT NULL DEFAULT 2,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Exchange rates table
CREATE TABLE IF NOT EXISTS exchange_rates (
    id TEXT PRIMARY KEY,
    from_currency TEXT NOT NULL,
    to_currency TEXT NOT NULL,
    rate REAL NOT NULL,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_currency, to_currency)
);

-- Expense comments table
CREATE TABLE IF NOT EXISTS expense_comments (
    id TEXT PRIMARY KEY,
    expense_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    FOREIGN KEY (expense_id) REFERENCES expenses(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Monthly summaries table
CREATE TABLE IF NOT EXISTS monthly_summaries (
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
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_currencies_code ON currencies(code);
CREATE INDEX IF NOT EXISTS idx_exchange_rates_currencies ON exchange_rates(from_currency, to_currency);
CREATE INDEX IF NOT EXISTS idx_expense_comments_expense ON expense_comments(expense_id);
CREATE INDEX IF NOT EXISTS idx_expense_comments_user ON expense_comments(user_id);
CREATE INDEX IF NOT EXISTS idx_expense_comments_deleted ON expense_comments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_monthly_summaries_user ON monthly_summaries(user_id);
CREATE INDEX IF NOT EXISTS idx_monthly_summaries_group ON monthly_summaries(group_id);
CREATE INDEX IF NOT EXISTS idx_monthly_summaries_year_month ON monthly_summaries(year, month);
CREATE INDEX IF NOT EXISTS idx_monthly_summaries_lookup ON monthly_summaries(user_id, group_id, year, month);
