package moddemod

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DeviceStatus struct {
	DeviceName string    `gorm:"column:device_name"`
	AlarmState string    `gorm:"column:alarm_state"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

type Repository interface {
	GetDownModulators() ([]DeviceStatus, error)
	GetDownDemodulators() ([]DeviceStatus, error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) GetDownModulators() ([]DeviceStatus, error) {
	var results []DeviceStatus
	err := r.db.Table("modulators").
		Select("device_name, alarm_state, updated_at").
		Where("status = ?", 0).
		Where("deleted_at IS NULL").
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("gagal query modulator: %w", err)
	}
	return results, nil
}

func (r *gormRepository) GetDownDemodulators() ([]DeviceStatus, error) {
	var results []DeviceStatus
	err := r.db.Table("demodulators").
		Select("device_name, alarm_state, updated_at").
		Where("status = ?", 0).
		Where("deleted_at IS NULL").
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("gagal query demodulator: %w", err)
	}
	return results, nil
}
