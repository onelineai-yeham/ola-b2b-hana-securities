package service

import (
	"context"

	"github.com/onelineai/hana-news-api/internal/model"
	"github.com/onelineai/hana-news-api/internal/repository"
)

// NewsService handles news query operations
type NewsService struct {
	goldRepo *repository.GoldRepository
}

func NewNewsService(goldRepo *repository.GoldRepository) *NewsService {
	return &NewsService{goldRepo: goldRepo}
}

// ListNews returns paginated news list
func (s *NewsService) ListNews(ctx context.Context, filter model.NewsFilter) (*model.NewsListResponse, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	items, total, err := s.goldRepo.ListNews(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &model.NewsListResponse{
		Data: items,
		Pagination: model.Pagination{
			Page:  filter.Page,
			Limit: filter.Limit,
			Total: total,
		},
	}, nil
}

// GetNewsDetail returns detailed news by ID
func (s *NewsService) GetNewsDetail(ctx context.Context, id string) (*model.NewsDetail, error) {
	return s.goldRepo.GetNewsDetail(ctx, id)
}
