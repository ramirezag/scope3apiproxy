package internal

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"scope3proxy/internal/cache"
	v2 "scope3proxy/internal/scope3/v2"
	"time"
)

const EmissionCacheKeySuffix = "_emission"

type EmissionService struct {
	logger          *zap.Logger
	scope3APIClient *v2.Scope3APIClient
	cache           *cache.Cache
	cacheTtl        time.Duration
}

type EmissionFilter struct {
	Country     string
	Channel     string
	InventoryId string
	Impressions int
	UtcDatetime string
	Priority    int
}

func NewEmissionService(
	logger *zap.Logger,
	scope3APIClient *v2.Scope3APIClient,
	cache *cache.Cache,
	cacheTtl time.Duration,
) *EmissionService {
	return &EmissionService{
		logger:          logger,
		scope3APIClient: scope3APIClient,
		cache:           cache,
		cacheTtl:        cacheTtl,
	}
}

type EmissionPerProperty map[string]interface{}

func (s *EmissionService) GetEmissions(filters []EmissionFilter) (*EmissionPerProperty, error) {
	result := EmissionPerProperty{}
	propertyPriorityMap := map[string]int{}

	var toFetchFromScope3 []v2.MeasureFilterRow
	for _, filter := range filters {
		propertyName := filter.InventoryId
		if emissions, exists := s.cache.Get(propertyName + EmissionCacheKeySuffix); exists {
			result[propertyName] = emissions
		} else {
			toFetchFromScope3 = append(toFetchFromScope3, v2.MeasureFilterRow{
				Country:     filter.Country,
				Channel:     filter.Channel,
				InventoryId: filter.InventoryId,
				Impressions: filter.Impressions,
				UtcDatetime: filter.UtcDatetime,
			})
			propertyPriorityMap[filter.InventoryId] = filter.Priority
		}
	}

	if len(toFetchFromScope3) > 0 {
		freshData, err := s.scope3APIClient.GetEmissionsBreakdown(&toFetchFromScope3)
		if err != nil {
			var serverError v2.Scope3ServerError
			if errors.As(err, &serverError) {
				// For any scope3 specific api server error (eg, server is down), the app will return whatever is in cache
				s.logger.Warn("Failed to fetch emissions breakdown from scope3 server.", zap.Error(err))
			} else {
				// might be application error
				return nil, fmt.Errorf("failed to fetch emissions breakdown from scope3 server: %w", err)
			}
		} else {
			for propertyName, emissions := range freshData {
				go func() {
					s.cache.Set(
						propertyName+EmissionCacheKeySuffix,
						emissions,
						propertyPriorityMap[propertyName],
						s.cacheTtl,
					)
				}()
				result[propertyName] = emissions
			}
		}
	}
	return &result, nil
}
