package satnet

import (
	"bella/internal/notifier"
	"bella/internal/types"
	"log"
	"time"

	"gorm.io/gorm"
)

type Satnet struct {
	Name          string
	FwdThroughput float64
	RtnThroughput float64
	Time          time.Time
}

type Service struct {
	repo     Repository
	notifier notifier.Notifier
	name     string
}

func NewService(dbFive *gorm.DB, notifier notifier.Notifier, name string) *Service {
	return &Service{
		repo:     NewGormRepository(dbFive),
		notifier: notifier,
		name:     name,
	}
}

func (s *Service) CheckAndAlert() {
	const thresholdKbps = 1000.0
	const alertThreshold = 3

	allData, err := s.repo.GetLastSatnetData()
	if err != nil {
		log.Printf("[%s] Error mendapatkan data satnet: %v", s.name, err)
		return
	}

	var degradedSatnetsForReport []types.SatnetDetail

	for _, data := range allData {
		if data.FwdThroughput < thresholdKbps {
			online, offline, err := s.repo.GetTerminalStatus(data.Name)
			if err != nil {
				log.Printf("[%s] Gagal mendapatkan status terminal untuk %s: %v", s.name, data.Name, err)
			}

			var totalAffected int64
			if online != nil {
				totalAffected += *online
			}
			if offline != nil {
				totalAffected += *offline
			}

			sendAlert := totalAffected > alertThreshold

			if sendAlert {
				startIssueTime, err := s.repo.GetStartIssueTime(data.Name)
				if err != nil {
					log.Printf("[%s] Gagal mendapatkan start issue time untuk %s: %v", s.name, data.Name, err)
				}

				degradedSatnetsForReport = append(degradedSatnetsForReport, types.SatnetDetail{
					Name:         data.Name,
					FwdTp:        data.FwdThroughput,
					RtnTp:        data.RtnThroughput,
					Time:         data.Time.Format(time.RFC3339),
					OnlineCount:  online,
					OfflineCount: offline,
					StartIssue:   startIssueTime,
				})
			}
		}
	}

	if len(degradedSatnetsForReport) > 0 {
		report := types.GatewayReport{
			FriendlyName: s.name,
			Satnets:      degradedSatnetsForReport,
		}
		log.Printf("[%s] Terdeteksi %d satnet memenuhi kriteria alert. Mengirim notifikasi...", s.name, len(degradedSatnetsForReport))
		err := s.notifier.SendSatnetAlert(report)
		if err != nil {
			log.Printf("[%s] Gagal mengirim notifikasi: %v", s.name, err)
		}
	} else {
		log.Printf("[%s] Semua satnet dalam kondisi normal atau di bawah ambang batas notifikasi.", s.name)
	}

}
