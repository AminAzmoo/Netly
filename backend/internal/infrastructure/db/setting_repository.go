package db

import (
    "context"
    "errors"

    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
    "gorm.io/gorm"
)

type systemSettingRepository struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewSystemSettingRepository(db *gorm.DB, log *logger.Logger) ports.SystemSettingRepository {
    return &systemSettingRepository{db: db, log: log}
}

func (r *systemSettingRepository) Get(ctx context.Context, key string) (*domain.SystemSetting, error) {
    var setting domain.SystemSetting
    if err := r.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        r.log.Errorw("setting_repo_get_failed", "key", key, "error", err)
        return nil, err
    }
    return &setting, nil
}

func (r *systemSettingRepository) Set(ctx context.Context, setting *domain.SystemSetting) error {
    var existing domain.SystemSetting
    err := r.db.WithContext(ctx).Where("key = ?", setting.Key).First(&existing).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            if err := r.db.WithContext(ctx).Create(setting).Error; err != nil {
                r.log.Errorw("setting_repo_create_failed", "key", setting.Key, "error", err)
                return err
            }
            r.log.Infow("setting_repo_create_ok", "key", setting.Key)
            return nil
        }
        r.log.Errorw("setting_repo_get_for_set_failed", "key", setting.Key, "error", err)
        return err
    }
    existing.Value = setting.Value
    existing.Type = setting.Type
    existing.Category = setting.Category
    if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
        r.log.Errorw("setting_repo_update_failed", "key", setting.Key, "error", err)
        return err
    }
    r.log.Infow("setting_repo_update_ok", "key", setting.Key)
    return nil
}

func (r *systemSettingRepository) GetByCategory(ctx context.Context, category string) ([]domain.SystemSetting, error) {
    var settings []domain.SystemSetting
    if err := r.db.WithContext(ctx).Where("category = ?", category).Find(&settings).Error; err != nil {
        r.log.Errorw("setting_repo_get_by_category_failed", "category", category, "error", err)
        return nil, err
    }
    r.log.Infow("setting_repo_get_by_category_ok", "category", category, "count", len(settings))
    return settings, nil
}

func (r *systemSettingRepository) Delete(ctx context.Context, key string) error {
    if err := r.db.WithContext(ctx).Where("key = ?", key).Delete(&domain.SystemSetting{}).Error; err != nil {
        r.log.Errorw("setting_repo_delete_failed", "key", key, "error", err)
        return err
    }
    r.log.Infow("setting_repo_delete_ok", "key", key)
    return nil
}
