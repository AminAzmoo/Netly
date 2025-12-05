package db

import (
	"github.com/netly/backend/internal/domain"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	// AutoMigrate all models
	err := db.AutoMigrate(
		&domain.Node{},
		&domain.Tunnel{},
		&domain.Service{},
		&domain.TimelineEvent{},
		&domain.SystemSetting{},
		&domain.IPAllocation{},
		&domain.PortAllocation{},
	)
	if err != nil {
		return err
	}

	// Create composite unique indexes
	if err := createCustomIndexes(db); err != nil {
		return err
	}

	return nil
}

func createCustomIndexes(db *gorm.DB) error {
	// Composite unique index for port allocation (node_id + port + protocol)
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_port_allocations_unique 
		ON port_allocations (node_id, port, protocol) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// Composite unique index for IP allocation (node_id + ip_address)
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_ip_allocations_unique 
		ON ip_allocations (node_id, ip_address) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// Index for timeline events querying by resource
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_timeline_events_resource 
		ON timeline_events (resource_type, resource_id) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	return nil
}
