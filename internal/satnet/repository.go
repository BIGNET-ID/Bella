package satnet

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	GetLastSatnetData() ([]Satnet, error)
	GetStartIssueTime(satnetName string) (*time.Time, error)
	GetTerminalStatus(satnetName string) (online *int64, offline *int64, err error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

type dbModel struct {
	SatnetName          string    `gorm:"column:satnet_name"`
	SatnetFwdThroughput float64   `gorm:"column:satnet_fwd_throughput"`
	SatnetRtnThroughput float64   `gorm:"column:satnet_rtn_throughput"`
	Time                time.Time `gorm:"column:time"`
}

func (dbModel) TableName() string { return "satnet_kpi" }

func (r *gormRepository) GetLastSatnetData() ([]Satnet, error) {
	var dbResults []dbModel
	sql := `
		SELECT DISTINCT ON (satnet_name)
			satnet_name,
			satnet_fwd_throughput,
			satnet_rtn_throughput,
			time
		FROM satnet_kpi
		ORDER BY satnet_name, time DESC;
	`
	if err := r.db.Raw(sql).Scan(&dbResults).Error; err != nil {
		return nil, fmt.Errorf("gagal query satnet_kpi: %w", err)
	}

	results := make([]Satnet, len(dbResults))
	for i, dbData := range dbResults {
		results[i] = Satnet{
			Name:          dbData.SatnetName,
			FwdThroughput: dbData.SatnetFwdThroughput,
			RtnThroughput: dbData.SatnetRtnThroughput,
			Time:          dbData.Time,
		}
	}
	return results, nil
}

func (r *gormRepository) GetStartIssueTime(satnetName string) (*time.Time, error) {
	var result struct {
		Time time.Time
	}
	sql := `
		SELECT time FROM satnet_kpi
		WHERE satnet_name = ? AND satnet_fwd_throughput < 1000
		AND time >= (
			SELECT time FROM satnet_kpi
			WHERE satnet_name = ? AND satnet_fwd_throughput >= 1000
			ORDER BY time DESC
			LIMIT 1
		)
		ORDER BY time ASC
		LIMIT 1;
	`
	err := r.db.Raw(sql, satnetName, satnetName).Scan(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if result.Time.IsZero() {
		return nil, nil
	}
	return &result.Time, nil
}

func (r *gormRepository) GetTerminalStatus(satnetName string) (*int64, *int64, error) {
	if r.db == nil {
		return nil, nil, fmt.Errorf("koneksi database (DB_FIVE) tidak tersedia")
	}

	var latestTime struct{ Time time.Time }
	err := r.db.Table("modem_kpi").
		Select("MAX(time) as time").
		Where("satnet = ? AND time > NOW() - INTERVAL '15 minutes'", satnetName).
		Scan(&latestTime).Error

	if err != nil || latestTime.Time.IsZero() {
		if err == gorm.ErrRecordNotFound || latestTime.Time.IsZero() {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("gagal mendapatkan waktu terakhir data terminal: %w", err)
	}

	var onlineCount, offlineCount int64
	r.db.Table("modem_kpi").Where("satnet = ? AND time = ? AND esno_avg > 0", satnetName, latestTime.Time).Count(&onlineCount)
	r.db.Table("modem_kpi").Where("satnet = ? AND time = ? AND (esno_avg <= 0 OR esno_avg IS NULL)", satnetName, latestTime.Time).Count(&offlineCount)

	return &onlineCount, &offlineCount, nil
}
