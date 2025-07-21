package satnet

import (
	"fmt"
	"log"
	"time"
	"bella/internal/notifier"

	"gorm.io/gorm"
)

type Satnet struct {
	Name          string
	FwdThroughput float64
	RtnThroughput float64
	Time          time.Time
}

type Repository interface {
	GetLastSatnetData() ([]Satnet, error)
}

type Service struct {
	repo     Repository
	notifier notifier.Notifier
	name     string
}

func NewService(db *gorm.DB, notifier notifier.Notifier, name string) *Service {
	return &Service{
		repo:     NewGormRepository(db),
		notifier: notifier,
		name:     name,
	}
}

func (s *Service) CheckAndAlert() {
	const thresholdKbps = 1000.0
	
	allData, err := s.repo.GetLastSatnetData()
	if err != nil {
		log.Printf("[%s] Error mendapatkan data satnet: %v", s.name, err)
		return
	}

	var degradedForAlert []notifier.DegradedSatnetInfo
	for _, data := range allData {
		if data.FwdThroughput < thresholdKbps || data.RtnThroughput < thresholdKbps {
			degradedForAlert = append(degradedForAlert, notifier.DegradedSatnetInfo{
				Name: data.Name,
				Fwd:  fmt.Sprintf("%.2f", data.FwdThroughput),
				Rtn:  fmt.Sprintf("%.2f", data.RtnThroughput),
			})
		}
	}

	if len(degradedForAlert) > 0 {
		log.Printf("[%s] Terdeteksi %d satnet bermasalah. Mengirim notifikasi...", s.name, len(degradedForAlert))
		err := s.notifier.SendSatnetAlert(s.name, degradedForAlert)
		if err != nil {
			log.Printf("[%s] Gagal mengirim notifikasi satnet: %v", s.name, err)
		}
	} else {
		log.Printf("[%s] Semua satnet dalam kondisi normal.", s.name)
	}
}