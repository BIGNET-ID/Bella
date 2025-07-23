package satnet

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

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
	err := r.db.Raw(sql).Scan(&dbResults).Error
	if err != nil {
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

type TerminalStatusRepository interface {
	GetTerminalStatus(satnetName string) (online *int64, offline *int64, err error)
}

type gormTerminalStatusRepository struct {
	db *gorm.DB
}

func NewGormTerminalStatusRepository(db *gorm.DB) TerminalStatusRepository {
	return &gormTerminalStatusRepository{db: db}
}

func (r *gormTerminalStatusRepository) GetTerminalStatus(satnetName string) (*int64, *int64, error) {
	if r.db == nil {
		return nil, nil, fmt.Errorf("koneksi database (DB_FIVE) tidak tersedia")
	}

	var recordCount int64
	if err := r.db.Table("modem_kpi").Where("satnet = ?", satnetName).Count(&recordCount).Error; err != nil {
		return nil, nil, fmt.Errorf("gagal pre-check data terminal: %w", err)
	}

	if recordCount == 0 {
		return nil, nil, nil
	}

	var latestTime struct{ Time time.Time }
	if err := r.db.Table("modem_kpi").Select("MAX(time) as time").Where("satnet = ?", satnetName).Scan(&latestTime).Error; err != nil {
		return nil, nil, fmt.Errorf("gagal mendapatkan waktu terakhir data terminal: %w", err)
	}

	var onlineCount, offlineCount int64
	r.db.Table("modem_kpi").Where("satnet = ? AND time = ? AND esno_avg > 0", satnetName, latestTime.Time).Count(&onlineCount)
	r.db.Table("modem_kpi").Where("satnet = ? AND time = ? AND (esno_avg <= 0 OR esno_avg IS NULL)", satnetName, latestTime.Time).Count(&offlineCount)

	return &onlineCount, &offlineCount, nil
}
