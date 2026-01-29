package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onelineai/hana-news-api/internal/model"
)

// SilverRepository handles read operations from silver schema
type SilverRepository struct {
	pool *pgxpool.Pool
}

func NewSilverRepository(pool *pgxpool.Pool) *SilverRepository {
	return &SilverRepository{pool: pool}
}

// GetJPMinkabuNewsSince returns JP Minkabu news updated since the given time
func (r *SilverRepository) GetJPMinkabuNewsSince(ctx context.Context, since *time.Time, limit int) ([]model.JPMinkabuNews, error) {
	var query string
	var args []interface{}

	if since == nil {
		query = `
			SELECT id, news_id, original_headline, original_story, 
			       translated_headline, translated_story, providers, topics, tickers,
			       creation_time, model_name, created_at, updated_at
			FROM silver.jp_minkabu_translated_news
			ORDER BY updated_at ASC
			LIMIT $1
		`
		args = []interface{}{limit}
	} else {
		query = `
			SELECT id, news_id, original_headline, original_story,
			       translated_headline, translated_story, providers, topics, tickers,
			       creation_time, model_name, created_at, updated_at
			FROM silver.jp_minkabu_translated_news
			WHERE updated_at > $1
			ORDER BY updated_at ASC
			LIMIT $2
		`
		args = []interface{}{since, limit}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanJPMinkabuNews(rows)
}

// GetCNWindNewsSince returns CN Wind news updated since the given time
func (r *SilverRepository) GetCNWindNewsSince(ctx context.Context, since *time.Time, limit int) ([]model.CNWindNews, error) {
	var query string
	var args []interface{}

	if since == nil {
		query = `
			SELECT id, object_id, original_title, original_content,
			       translated_title, translated_content, publish_date, source,
			       sections, wind_codes, keywords, model_name, created_at, updated_at
			FROM silver.cn_wind_translated_news
			ORDER BY updated_at ASC
			LIMIT $1
		`
		args = []interface{}{limit}
	} else {
		query = `
			SELECT id, object_id, original_title, original_content,
			       translated_title, translated_content, publish_date, source,
			       sections, wind_codes, keywords, model_name, created_at, updated_at
			FROM silver.cn_wind_translated_news
			WHERE updated_at > $1
			ORDER BY updated_at ASC
			LIMIT $2
		`
		args = []interface{}{since, limit}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCNWindNews(rows)
}

func scanJPMinkabuNews(rows pgx.Rows) ([]model.JPMinkabuNews, error) {
	var news []model.JPMinkabuNews
	for rows.Next() {
		var n model.JPMinkabuNews
		err := rows.Scan(
			&n.ID, &n.NewsID, &n.OriginalHeadline, &n.OriginalStory,
			&n.TranslatedHeadline, &n.TranslatedStory, &n.Providers, &n.Topics, &n.Tickers,
			&n.CreationTime, &n.ModelName, &n.CreatedAt, &n.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		news = append(news, n)
	}
	return news, rows.Err()
}

func scanCNWindNews(rows pgx.Rows) ([]model.CNWindNews, error) {
	var news []model.CNWindNews
	for rows.Next() {
		var n model.CNWindNews
		err := rows.Scan(
			&n.ID, &n.ObjectID, &n.OriginalTitle, &n.OriginalContent,
			&n.TranslatedTitle, &n.TranslatedContent, &n.PublishDate, &n.Source,
			&n.Sections, &n.WindCodes, &n.Keywords, &n.ModelName, &n.CreatedAt, &n.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		news = append(news, n)
	}
	return news, rows.Err()
}
