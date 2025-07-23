package satnet

import (
	"log"
	"time"
	"bella/internal/notifier"
	"bella/internal/types"

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
	repo         Repository
	terminalRepo TerminalStatusRepository
	notifier     notifier.Notifier
	name         string
}

func NewService(dbFive *gorm.DB, notifier notifier.Notifier, name string) *Service {
	return &Service{
		repo:         NewGormRepository(dbFive),
		terminalRepo: NewGormTerminalStatusRepository(dbFive),
		notifier:     notifier,
		name:         name,
	}
}

func (s *Service) CheckAndAlert() {
	const thresholdKbps = 1000.0

	allData, err := s.repo.GetLastSatnetData()
	if err != nil {
		log.Printf("[%s] Error mendapatkan data satnet: %v", s.name, err)
		return
	}

	var degradedSatnets []types.SatnetDetail

	for _, data := range allData {
		if data.FwdThroughput < thresholdKbps {
			online, offline, err := s.terminalRepo.GetTerminalStatus(data.Name)
			if err != nil {
				log.Printf("[%s] Gagal mendapatkan status terminal untuk %s: %v", s.name, data.Name, err)
			}

			degradedSatnets = append(degradedSatnets, types.SatnetDetail{
				Name:         data.Name,
				FwdTp:        data.FwdThroughput,
				RtnTp:        data.RtnThroughput,
				Time:         data.Time.Format(time.RFC3339),
				OnlineCount:  online,
				OfflineCount: offline,
			})
		}
	}

	if len(degradedSatnets) > 0 {
		report := types.GatewayReport{
			FriendlyName: s.name,
			Satnets:      degradedSatnets,
		}
		log.Printf("[%s] Terdeteksi %d satnet bermasalah. Mengirim notifikasi...", s.name, len(degradedSatnets))
		err := s.notifier.SendSatnetAlert(report)
		if err != nil {
			log.Printf("[%s] Gagal mengirim notifikasi: %v", s.name, err)
		}
	} else {
		log.Printf("[%s] Semua satnet dalam kondisi normal.", s.name)
	}
}
