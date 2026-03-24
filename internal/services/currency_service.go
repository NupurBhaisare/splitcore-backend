package services

import (
	"github.com/splitcore/backend/internal/database"
	"github.com/splitcore/backend/internal/models"
)

type CurrencyService struct{}

func NewCurrencyService() *CurrencyService {
	return &CurrencyService{}
}

// GetAllCurrencies returns all supported currencies.
func (s *CurrencyService) GetAllCurrencies() ([]models.Currency, error) {
	rows, err := database.DB.Query(
		`SELECT code, name, symbol, decimal_places, updated_at FROM currencies ORDER BY code`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var currencies []models.Currency
	for rows.Next() {
		var c models.Currency
		if err := rows.Scan(&c.Code, &c.Name, &c.Symbol, &c.DecimalPlaces, &c.UpdatedAt); err != nil {
			return nil, err
		}
		currencies = append(currencies, c)
	}

	return currencies, nil
}

// GetExchangeRates returns all exchange rates.
func (s *CurrencyService) GetExchangeRates() ([]models.ExchangeRate, error) {
	rows, err := database.DB.Query(
		`SELECT id, from_currency, to_currency, rate, fetched_at FROM exchange_rates ORDER BY from_currency, to_currency`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []models.ExchangeRate
	for rows.Next() {
		var r models.ExchangeRate
		if err := rows.Scan(&r.ID, &r.FromCurrency, &r.ToCurrency, &r.Rate, &r.FetchedAt); err != nil {
			return nil, err
		}
		rates = append(rates, r)
	}

	return rates, nil
}

// ConvertCurrency converts an amount from one currency to another.
func (s *CurrencyService) ConvertCurrency(amountCents int64, fromCurrency, toCurrency string) (int64, error) {
	if fromCurrency == toCurrency {
		return amountCents, nil
	}

	var rate float64
	err := database.DB.QueryRow(
		`SELECT rate FROM exchange_rates WHERE from_currency = ? AND to_currency = ?`,
		fromCurrency, toCurrency,
	).Scan(&rate)

	if err != nil {
		// Try reverse rate
		err = database.DB.QueryRow(
			`SELECT 1.0 / rate FROM exchange_rates WHERE from_currency = ? AND to_currency = ?`,
			toCurrency, fromCurrency,
		).Scan(&rate)
		if err != nil {
			return amountCents, nil // Fallback: return original amount
		}
	}

	// Convert: amount_cents is in the "from" currency's smallest unit
	// We need to account for decimal places difference
	var fromDecimals, toDecimals int
	database.DB.QueryRow(`SELECT decimal_places FROM currencies WHERE code = ?`, fromCurrency).Scan(&fromDecimals)
	database.DB.QueryRow(`SELECT decimal_places FROM currencies WHERE code = ?`, toCurrency).Scan(&toDecimals)

	// Convert to float, apply rate, convert back to cents
	floatAmount := float64(amountCents)
	converted := floatAmount * rate
	result := int64(converted)
	return result, nil
}

// GetCurrency returns a single currency by code.
func (s *CurrencyService) GetCurrency(code string) (*models.Currency, error) {
	var c models.Currency
	err := database.DB.QueryRow(
		`SELECT code, name, symbol, decimal_places, updated_at FROM currencies WHERE code = ?`,
		code,
	).Scan(&c.Code, &c.Name, &c.Symbol, &c.DecimalPlaces, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
