package services

import "context"

// UsageAggregator is responsible for collecting and aggregating usage stats.
type UsageAggregator struct{}

func NewUsageAggregator() *UsageAggregator {
	return &UsageAggregator{}
}

func (u *UsageAggregator) Aggregate(ctx context.Context, nodeID uint) error {
	return nil
}
