package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/onelineai/hana-news-api/internal/db"
	"github.com/onelineai/hana-news-api/internal/model"
	"github.com/onelineai/hana-news-api/internal/service"
)

type Handler struct {
	newsService *service.NewsService
	db          *db.DB
	logger      *slog.Logger
}

func New(newsService *service.NewsService, db *db.DB, logger *slog.Logger) *Handler {
	return &Handler{
		newsService: newsService,
		db:          db,
		logger:      logger,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	})
	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
	))

	r.Get("/health", h.healthCheck)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/news/{country}", h.listNewsByCountry)
		r.Get("/news/{country}/{exchange}", h.listNewsByCountryExchange)
	})

	return r
}

// healthCheck godoc
// @Summary      Health check
// @Description  Check service health status
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Router       /health [get]
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.db.HealthCheck(r.Context()); err != nil {
		h.logger.Error("health check failed", "error", err)
		h.respondError(w, http.StatusServiceUnavailable, "service unhealthy")
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// listNewsByCountry godoc
// @Summary      List news by country
// @Description  Get paginated list of translated news articles for a specific country
// @Tags         news
// @Accept       json
// @Produce      json
// @Param        country path      string  true   "Country code (JP or CN)"
// @Param        ticker  query     string  false  "Filter by ticker/stock code"
// @Param        from    query     string  false  "Start time (RFC3339 format)"
// @Param        to      query     string  false  "End time (RFC3339 format)"
// @Param        page    query     int     false  "Page number (default: 1)"
// @Param        limit   query     int     false  "Items per page (default: 20, max: 100)"
// @Success      200     {object}  model.NewsListResponse
// @Failure      400     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /v1/news/{country} [get]
func (h *Handler) listNewsByCountry(w http.ResponseWriter, r *http.Request) {
	country := strings.ToUpper(chi.URLParam(r, "country"))

	c := model.CountryCode(country)
	if c != model.CountryJP && c != model.CountryCN {
		h.respondError(w, http.StatusBadRequest, "invalid country, must be 'JP' or 'CN'")
		return
	}

	filter := h.parseCommonFilters(r)
	source := c.ToNewsSource()
	filter.Source = &source

	if ticker := r.URL.Query().Get("ticker"); ticker != "" {
		filter.Ticker = &ticker
	}

	h.executeListNews(w, r, filter)
}

// listNewsByCountryExchange godoc
// @Summary      List news by country and exchange
// @Description  Get paginated list of translated news articles for a specific country and exchange
// @Tags         news
// @Accept       json
// @Produce      json
// @Param        country  path      string  true   "Country code (CN)"
// @Param        exchange path      string  true   "Exchange code (HK, SH, SZ, BJ)"
// @Param        ticker   query     string  false  "Filter by ticker code (without exchange suffix)"
// @Param        from     query     string  false  "Start time (RFC3339 format)"
// @Param        to       query     string  false  "End time (RFC3339 format)"
// @Param        page     query     int     false  "Page number (default: 1)"
// @Param        limit    query     int     false  "Items per page (default: 20, max: 100)"
// @Success      200      {object}  model.NewsListResponse
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /v1/news/{country}/{exchange} [get]
func (h *Handler) listNewsByCountryExchange(w http.ResponseWriter, r *http.Request) {
	country := strings.ToUpper(chi.URLParam(r, "country"))
	exchange := strings.ToUpper(chi.URLParam(r, "exchange"))

	if country != "CN" {
		h.respondError(w, http.StatusBadRequest, "exchange filter only supported for CN")
		return
	}

	validExchanges := map[string]bool{"HK": true, "SH": true, "SZ": true, "BJ": true}
	if !validExchanges[exchange] {
		h.respondError(w, http.StatusBadRequest, "invalid exchange, must be 'HK', 'SH', 'SZ', or 'BJ'")
		return
	}

	filter := h.parseCommonFilters(r)
	source := model.SourceCNWind
	filter.Source = &source
	filter.Exchange = &exchange

	if ticker := r.URL.Query().Get("ticker"); ticker != "" {
		fullTicker := fmt.Sprintf("%s.%s", ticker, exchange)
		filter.Ticker = &fullTicker
	}

	h.executeListNews(w, r, filter)
}

func (h *Handler) parseCommonFilters(r *http.Request) model.NewsFilter {
	filter := model.NewsFilter{
		Page:  1,
		Limit: 20,
	}

	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}

	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filter.Page = p
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	return filter
}

func (h *Handler) executeListNews(w http.ResponseWriter, r *http.Request, filter model.NewsFilter) {
	resp, err := h.newsService.ListNews(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to list news", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	h.respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
