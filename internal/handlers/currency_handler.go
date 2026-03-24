package handlers

import (
	"net/http"

	"github.com/splitcore/backend/internal/services"
	"github.com/splitcore/backend/pkg/utils"
)

type CurrencyHandler struct {
	currencyService *services.CurrencyService
}

func NewCurrencyHandler() *CurrencyHandler {
	return &CurrencyHandler{
		currencyService: services.NewCurrencyService(),
	}
}

func (h *CurrencyHandler) GetCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies, err := h.currencyService.GetAllCurrencies()
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, http.StatusOK, currencies)
}

func (h *CurrencyHandler) GetExchangeRates(w http.ResponseWriter, r *http.Request) {
	rates, err := h.currencyService.GetExchangeRates()
	if err != nil {
		utils.InternalError(w, err.Error())
		return
	}
	utils.Success(w, http.StatusOK, rates)
}
