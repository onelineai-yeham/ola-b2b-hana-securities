package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/onelineai/hana-news-api/internal/model"
	"github.com/onelineai/hana-news-api/internal/repository"
)

const batchSize = 500

// BatchService handles ETL batch operations
type BatchService struct {
	silverRepo *repository.SilverRepository
	goldRepo   *repository.GoldRepository
	logger     *slog.Logger
}

func NewBatchService(silverRepo *repository.SilverRepository, goldRepo *repository.GoldRepository, logger *slog.Logger) *BatchService {
	return &BatchService{
		silverRepo: silverRepo,
		goldRepo:   goldRepo,
		logger:     logger,
	}
}

// SyncAll synchronizes all news sources from silver to gold
func (s *BatchService) SyncAll(ctx context.Context) error {
	s.logger.Info("starting batch sync")
	start := time.Now()

	// Sync JP Minkabu news
	jpCount, err := s.syncJPMinkabu(ctx)
	if err != nil {
		s.logger.Error("failed to sync JP Minkabu news", "error", err)
		return err
	}

	// Sync CN Wind news
	cnCount, err := s.syncCNWind(ctx)
	if err != nil {
		s.logger.Error("failed to sync CN Wind news", "error", err)
		return err
	}

	s.logger.Info("batch sync completed",
		"duration", time.Since(start),
		"jp_minkabu_count", jpCount,
		"cn_wind_count", cnCount,
	)

	return nil
}

func (s *BatchService) syncJPMinkabu(ctx context.Context) (int, error) {
	source := model.SourceJPMinkabu

	// Get last sync time
	lastSync, err := s.goldRepo.GetLastSyncTime(ctx, source)
	if err != nil {
		return 0, err
	}

	totalSynced := 0
	var lastUpdatedAt time.Time

	for {
		// Fetch batch from silver
		news, err := s.silverRepo.GetJPMinkabuNewsSince(ctx, lastSync, batchSize)
		if err != nil {
			return totalSynced, err
		}

		if len(news) == 0 {
			break
		}

		s.logger.Debug("fetched JP Minkabu news batch", "count", len(news))

		// Convert to unified format
		unifiedNews := make([]*model.TranslatedNews, len(news))
		for i, n := range news {
			unifiedNews[i] = n.ToTranslatedNews()
		}

		// Upsert to gold
		affected, err := s.goldRepo.UpsertNews(ctx, unifiedNews)
		if err != nil {
			return totalSynced, err
		}

		totalSynced += affected
		lastUpdatedAt = news[len(news)-1].UpdatedAt

		// Update last sync pointer for next iteration
		lastSync = &lastUpdatedAt

		// If we got less than batch size, we're done
		if len(news) < batchSize {
			break
		}
	}

	// Update sync metadata
	if totalSynced > 0 {
		if err := s.goldRepo.UpdateSyncMetadata(ctx, source, lastUpdatedAt, totalSynced); err != nil {
			s.logger.Warn("failed to update sync metadata", "source", source, "error", err)
		}
	}

	return totalSynced, nil
}

func (s *BatchService) syncCNWind(ctx context.Context) (int, error) {
	source := model.SourceCNWind

	// Get last sync time
	lastSync, err := s.goldRepo.GetLastSyncTime(ctx, source)
	if err != nil {
		return 0, err
	}

	totalSynced := 0
	var lastUpdatedAt time.Time

	for {
		// Fetch batch from silver
		news, err := s.silverRepo.GetCNWindNewsSince(ctx, lastSync, batchSize)
		if err != nil {
			return totalSynced, err
		}

		if len(news) == 0 {
			break
		}

		s.logger.Debug("fetched CN Wind news batch", "count", len(news))

		// Convert to unified format
		unifiedNews := make([]*model.TranslatedNews, len(news))
		for i, n := range news {
			unifiedNews[i] = n.ToTranslatedNews()
		}

		// Upsert to gold
		affected, err := s.goldRepo.UpsertNews(ctx, unifiedNews)
		if err != nil {
			return totalSynced, err
		}

		totalSynced += affected
		lastUpdatedAt = news[len(news)-1].UpdatedAt

		// Update last sync pointer for next iteration
		lastSync = &lastUpdatedAt

		// If we got less than batch size, we're done
		if len(news) < batchSize {
			break
		}
	}

	// Update sync metadata
	if totalSynced > 0 {
		if err := s.goldRepo.UpdateSyncMetadata(ctx, source, lastUpdatedAt, totalSynced); err != nil {
			s.logger.Warn("failed to update sync metadata", "source", source, "error", err)
		}
	}

	return totalSynced, nil
}
