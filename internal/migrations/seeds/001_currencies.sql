-- Seed: Common currencies

INSERT OR IGNORE INTO currencies (code, name, symbol, decimal_places, updated_at) VALUES
    ('USD', 'US Dollar', '$', 2, CURRENT_TIMESTAMP),
    ('EUR', 'Euro', '€', 2, CURRENT_TIMESTAMP),
    ('GBP', 'British Pound', '£', 2, CURRENT_TIMESTAMP),
    ('JPY', 'Japanese Yen', '¥', 0, CURRENT_TIMESTAMP),
    ('CAD', 'Canadian Dollar', 'C$', 2, CURRENT_TIMESTAMP),
    ('AUD', 'Australian Dollar', 'A$', 2, CURRENT_TIMESTAMP),
    ('CHF', 'Swiss Franc', 'CHF', 2, CURRENT_TIMESTAMP),
    ('CNY', 'Chinese Yuan', '¥', 2, CURRENT_TIMESTAMP),
    ('INR', 'Indian Rupee', '₹', 2, CURRENT_TIMESTAMP),
    ('MXN', 'Mexican Peso', 'MX$', 2, CURRENT_TIMESTAMP),
    ('BRL', 'Brazilian Real', 'R$', 2, CURRENT_TIMESTAMP),
    ('KRW', 'South Korean Won', '₩', 0, CURRENT_TIMESTAMP),
    ('SGD', 'Singapore Dollar', 'S$', 2, CURRENT_TIMESTAMP),
    ('HKD', 'Hong Kong Dollar', 'HK$', 2, CURRENT_TIMESTAMP),
    ('NOK', 'Norwegian Krone', 'kr', 2, CURRENT_TIMESTAMP),
    ('SEK', 'Swedish Krona', 'kr', 2, CURRENT_TIMESTAMP),
    ('DKK', 'Danish Krone', 'kr', 2, CURRENT_TIMESTAMP),
    ('NZD', 'New Zealand Dollar', 'NZ$', 2, CURRENT_TIMESTAMP),
    ('ZAR', 'South African Rand', 'R', 2, CURRENT_TIMESTAMP),
    ('THB', 'Thai Baht', '฿', 2, CURRENT_TIMESTAMP);
