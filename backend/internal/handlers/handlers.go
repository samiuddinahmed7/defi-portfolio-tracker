// Package handlers implements the HTTP handlers for the DeFi portfolio tracker
// REST API. Handlers are intentionally thin: they decode inputs, call the
// appropriate service, and encode the response.
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/models"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/services"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/validation"
)

// Handler groups all HTTP handlers and their shared dependencies.
type Handler struct {
	portfolio *services.PortfolioService
	price     *services.PriceService
	log       *slog.Logger
	isDemo    bool
}

// New returns a Handler.
func New(
	portfolio *services.PortfolioService,
	price *services.PriceService,
	isDemo bool,
	log *slog.Logger,
) *Handler {
	return &Handler{
		portfolio: portfolio,
		price:     price,
		isDemo:    isDemo,
		log:       log.With("component", "handler"),
	}
}

// — Route registration -------------------------------------------------------

// RegisterRoutes wires up all API routes on mux.
// We use Go 1.22 enhanced ServeMux which supports method and path parameters.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.Health)
	mux.HandleFunc("GET /api/v1/portfolio/{address}", h.GetPortfolio)
	mux.HandleFunc("GET /api/v1/portfolio/{address}/balance", h.GetNativeBalance)
	mux.HandleFunc("GET /api/v1/portfolio/{address}/tokens", h.GetTokenBalances)
	mux.HandleFunc("GET /api/v1/portfolio/{address}/transactions", h.GetTransactions)
	mux.HandleFunc("GET /api/v1/prices/{symbol}", h.GetPrice)
}

// — Handlers -----------------------------------------------------------------

// Health returns a simple health-check response.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
		"demo":    h.isDemo,
	})
}

// GetPortfolio returns the full portfolio summary for a wallet address.
func (h *Handler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if err := validation.ValidateAddress(address); err != nil {
		respondError(w, http.StatusBadRequest, "invalid Ethereum address")
		return
	}

	summary, err := h.portfolio.GetPortfolio(r.Context(), address)
	if err != nil {
		var addrErr validation.ErrInvalidAddress
		if errors.As(err, &addrErr) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.log.Error("get portfolio failed", "address", address, "err", err)
		respondError(w, http.StatusBadGateway, "failed to fetch portfolio data")
		return
	}

	respond(w, http.StatusOK, summary)
}

// GetNativeBalance returns only the ETH balance for a wallet.
func (h *Handler) GetNativeBalance(w http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if err := validation.ValidateAddress(address); err != nil {
		respondError(w, http.StatusBadRequest, "invalid Ethereum address")
		return
	}

	bal, err := h.portfolio.GetNativeBalance(r.Context(), address)
	if err != nil {
		h.log.Error("get native balance failed", "address", address, "err", err)
		respondError(w, http.StatusBadGateway, "failed to fetch balance")
		return
	}

	respond(w, http.StatusOK, bal)
}

// GetTokenBalances returns ERC-20 holdings for a wallet.
func (h *Handler) GetTokenBalances(w http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if err := validation.ValidateAddress(address); err != nil {
		respondError(w, http.StatusBadRequest, "invalid Ethereum address")
		return
	}

	balances, err := h.portfolio.GetTokenBalances(r.Context(), address)
	if err != nil {
		h.log.Error("get token balances failed", "address", address, "err", err)
		respondError(w, http.StatusBadGateway, "failed to fetch token balances")
		return
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"address":  strings.ToLower(address),
		"tokens":   balances,
		"count":    len(balances),
	})
}

// GetTransactions returns a paginated list of transactions for a wallet.
func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if err := validation.ValidateAddress(address); err != nil {
		respondError(w, http.StatusBadRequest, "invalid Ethereum address")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	txPage, err := h.portfolio.GetTransactions(r.Context(), address, page, perPage)
	if err != nil {
		h.log.Error("get transactions failed", "address", address, "err", err)
		respondError(w, http.StatusBadGateway, "failed to fetch transactions")
		return
	}

	respond(w, http.StatusOK, txPage)
}

// GetPrice returns the current price for a token symbol.
func (h *Handler) GetPrice(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(r.PathValue("symbol"))
	if symbol == "" {
		respondError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	priceResp, err := h.price.GetPrice(r.Context(), symbol)
	if err != nil {
		h.log.Error("get price failed", "symbol", symbol, "err", err)
		respondError(w, http.StatusBadGateway, "failed to fetch price")
		return
	}

	respond(w, http.StatusOK, priceResp)
}

// — Response helpers ---------------------------------------------------------

func respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Nothing useful to do at this point — headers already sent.
		return
	}
}

func respondError(w http.ResponseWriter, code int, message string) {
	respond(w, code, models.ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}
