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
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (dbModel) TableName() string { return "satnets" }

func (r *gormRepository) GetLastSatnetData() ([]Satnet, error) {
	var dbResults []dbModel
	sql := `
		SELECT DISTINCT ON (satnet_name)
			satnet_name,
			satnet_fwd_throughput,
			satnet_rtn_throughput,
			updated_at AS time
		FROM satnets
		ORDER BY satnet_name, time DESC;
	`
	err := r.db.Raw(sql).Scan(&dbResults).Error
	if err != nil {
		return nil, fmt.Errorf("gagal query satnet throughput: %w", err)
	}

	results := make([]Satnet, len(dbResults))
	for i, dbData := range dbResults {
		results[i] = Satnet{
			Name:          dbData.SatnetName,
			FwdThroughput: dbData.SatnetFwdThroughput,
			RtnThroughput: dbData.SatnetRtnThroughput,
			Time:          dbData.UpdatedAt,
		}
	}
	return results, nil
}