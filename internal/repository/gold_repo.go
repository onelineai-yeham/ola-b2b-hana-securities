package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onelineai/hana-news-api/internal/model"
)

// GoldRepository handles read/write operations for gold schema
type GoldRepository struct {
	pool *pgxpool.Pool
}

func NewGoldRepository(pool *pgxpool.Pool) *GoldRepository {
	return &GoldRepository{pool: pool}
}

// GetLastSyncTime returns the last sync time for a source
func (r *GoldRepository) GetLastSyncTime(ctx context.Context, source model.NewsSource) (*time.Time, error) {
	var lastSyncedAt *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT last_synced_at FROM gold.sync_metadata WHERE source = $1`,
		string(source),
	).Scan(&lastSyncedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return lastSyncedAt, err
}

// UpdateSyncMetadata updates the sync metadata for a source
func (r *GoldRepository) UpdateSyncMetadata(ctx context.Context, source model.NewsSource, syncedAt time.Time, count int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE gold.sync_metadata 
		SET last_synced_at = $1, last_sync_count = $2, updated_at = NOW()
		WHERE source = $3
	`, syncedAt, count, string(source))
	return err
}

// UpsertNews upserts translated news records into the unified table
func (r *GoldRepository) UpsertNews(ctx context.Context, news []*model.TranslatedNews) (int, error) {
	if len(news) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, n := range news {
		batch.Queue(`
			INSERT INTO gold.translated_news 
				(source, source_news_id, original_headline, original_content,
				 translated_headline, translated_content, tickers, topics, keywords,
				 provider, published_at, model_name, source_created_at, source_updated_at, synced_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
			ON CONFLICT (source, source_news_id) DO UPDATE SET
				original_headline = EXCLUDED.original_headline,
				original_content = EXCLUDED.original_content,
				translated_headline = EXCLUDED.translated_headline,
				translated_content = EXCLUDED.translated_content,
				tickers = EXCLUDED.tickers,
				topics = EXCLUDED.topics,
				keywords = EXCLUDED.keywords,
				provider = EXCLUDED.provider,
				published_at = EXCLUDED.published_at,
				model_name = EXCLUDED.model_name,
				source_updated_at = EXCLUDED.source_updated_at,
				synced_at = NOW()
		`, n.Source, n.SourceNewsID, n.OriginalHeadline, n.OriginalContent,
			n.TranslatedHeadline, n.TranslatedContent, n.Tickers, n.Topics, n.Keywords,
			n.Provider, n.PublishedAt, n.ModelName, n.SourceCreatedAt, n.SourceUpdatedAt)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	affected := 0
	for range news {
		ct, err := results.Exec()
		if err != nil {
			return affected, err
		}
		affected += int(ct.RowsAffected())
	}
	return affected, nil
}

// ListNews returns paginated news list with optional filtering
func (r *GoldRepository) ListNews(ctx context.Context, filter model.NewsFilter) ([]model.NewsListItem, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Source != nil {
		conditions = append(conditions, fmt.Sprintf("source = $%d", argIdx))
		args = append(args, string(*filter.Source))
		argIdx++
	}

	if filter.Ticker != nil {
		conditions = append(conditions, fmt.Sprintf("$%d = ANY(tickers)", argIdx))
		args = append(args, *filter.Ticker)
		argIdx++
	}

	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("published_at >= $%d", argIdx))
		args = append(args, *filter.From)
		argIdx++
	}

	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("published_at <= $%d", argIdx))
		args = append(args, *filter.To)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM gold.translated_news %s`, whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query with pagination
	offset := (filter.Page - 1) * filter.Limit
	dataArgs := append(args, filter.Limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT source, source_news_id, translated_headline, tickers, topics, published_at
		FROM gold.translated_news
		%s
		ORDER BY published_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []model.NewsListItem
	for rows.Next() {
		var item model.NewsListItem
		var source string
		var sourceNewsID string
		if err := rows.Scan(&source, &sourceNewsID, &item.Headline, &item.Tickers, &item.Topics, &item.PublishedAt); err != nil {
			return nil, 0, err
		}
		item.Source = model.NewsSource(source)
		item.ID = fmt.Sprintf("%s_%s", source, sourceNewsID)
		items = append(items, item)
	}

	return items, total, rows.Err()
}

// GetNewsDetail returns detailed news by ID
func (r *GoldRepository) GetNewsDetail(ctx context.Context, id string) (*model.NewsDetail, error) {
	// Parse ID: format is "source_sourceNewsId"
	parts := strings.SplitN(id, "_", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid news id format: %s", id)
	}

	source := parts[0]
	// Handle cases like "jp_minkabu_xxx" where source itself has underscore
	if source == "jp" && len(parts[1]) > 0 {
		subParts := strings.SplitN(parts[1], "_", 2)
		if len(subParts) == 2 && subParts[0] == "minkabu" {
			source = "jp_minkabu"
			parts[1] = subParts[1]
		}
	} else if source == "cn" && len(parts[1]) > 0 {
		subParts := strings.SplitN(parts[1], "_", 2)
		if len(subParts) == 2 && subParts[0] == "wind" {
			source = "cn_wind"
			parts[1] = subParts[1]
		}
	}
	sourceNewsID := parts[1]

	var detail model.NewsDetail
	var sourceStr string

	err := r.pool.QueryRow(ctx, `
		SELECT source, source_news_id, original_headline, original_content,
		       translated_headline, translated_content, tickers, topics, keywords,
		       published_at, provider, model_name
		FROM gold.translated_news
		WHERE source = $1 AND source_news_id = $2
	`, source, sourceNewsID).Scan(
		&sourceStr, &detail.ID, &detail.OriginalHeadline, &detail.OriginalContent,
		&detail.TranslatedHeadline, &detail.TranslatedContent, &detail.Tickers, &detail.Topics, &detail.Keywords,
		&detail.PublishedAt, &detail.Provider, &detail.ModelName,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	detail.Source = model.NewsSource(sourceStr)
	detail.ID = fmt.Sprintf("%s_%s", sourceStr, detail.ID)

	return &detail, nil
}
