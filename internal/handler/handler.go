package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/onelineai/hana-news-api/internal/db"
	"github.com/onelineai/hana-news-api/internal/model"
	"github.com/onelineai/hana-news-api/internal/service"
)

// Handler handles HTTP requests
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

// Router returns the HTTP router
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Swagger docs
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	})
	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
	))

	// Health check
	r.Get("/health", h.healthCheck)

	// API v1
	r.Route("/v1", func(r chi.Router) {
		r.Get("/news", h.listNews)
		r.Get("/news/{id}", h.getNewsDetail)
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

// listNews godoc
// @Summary      List news
// @Description  Get paginated list of translated news articles
// @Tags         news
// @Accept       json
// @Produce      json
// @Param        source  query     string  false  "News source filter (jp_minkabu or cn_wind)"
// @Param        ticker  query     string  false  "Filter by ticker/stock code"
// @Param        from    query     string  false  "Start time (RFC3339 format)"
// @Param        to      query     string  false  "End time (RFC3339 format)"
// @Param        page    query     int     false  "Page number (default: 1)"
// @Param        limit   query     int     false  "Items per page (default: 20, max: 100)"
// @Success      200     {object}  model.NewsListResponse
// @Failure      400     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /v1/news [get]
func (h *Handler) listNews(w http.ResponseWriter, r *http.Request) {
	filter := model.NewsFilter{
		Page:  1,
		Limit: 20,
	}

	// Parse query params
	if source := r.URL.Query().Get("source"); source != "" {
		s := model.NewsSource(source)
		if s != model.SourceJPMinkabu && s != model.SourceCNWind {
			h.respondError(w, http.StatusBadRequest, "invalid source, must be 'jp_minkabu' or 'cn_wind'")
			return
		}
		filter.Source = &s
	}

	if ticker := r.URL.Query().Get("ticker"); ticker != "" {
		filter.Ticker = &ticker
	}

	if from := r.URL.Query().Get("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid 'from' format, use RFC3339")
			return
		}
		filter.From = &t
	}

	if to := r.URL.Query().Get("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid 'to' format, use RFC3339")
			return
		}
		filter.To = &t
	}

	if page := r.URL.Query().Get("page"); page != "" {
		p, err := strconv.Atoi(page)
		if err != nil || p < 1 {
			h.respondError(w, http.StatusBadRequest, "invalid 'page' parameter")
			return
		}
		filter.Page = p
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		l, err := strconv.Atoi(limit)
		if err != nil || l < 1 || l > 100 {
			h.respondError(w, http.StatusBadRequest, "invalid 'limit' parameter (1-100)")
			return
		}
		filter.Limit = l
	}

	resp, err := h.newsService.ListNews(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to list news", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// getNewsDetail godoc
// @Summary      Get news detail
// @Description  Get detailed information of a specific news article
// @Tags         news
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "News ID (e.g., jp_minkabu_12345 or cn_wind_abc123)"
// @Success      200  {object}  model.NewsDetail
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /v1/news/{id} [get]
func (h *Handler) getNewsDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "missing news id")
		return
	}

	detail, err := h.newsService.GetNewsDetail(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get news detail", "id", id, "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if detail == nil {
		h.respondError(w, http.StatusNotFound, "news not found")
		return
	}

	h.respondJSON(w, http.StatusOK, detail)
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
