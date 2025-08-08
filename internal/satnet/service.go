package satnet

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"bella/internal/notifier"
	"bella/internal/state"
	"bella/internal/types"

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
	state    *state.Manager
	name     string
}

func NewService(dbFive *gorm.DB, notifier notifier.Notifier, stateMgr *state.Manager, name string) *Service {
	return &Service{
		repo:     NewGormRepository(dbFive),
		notifier: notifier,
		state:    stateMgr,
		name:     name,
	}
}

func ParseStringToWIB(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Now(), nil
	}
	const layout = "2006-01-02T15:04:05"
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.Time{}, err
	}
	if len(timeStr) < 19 {
		return time.Time{}, fmt.Errorf("string waktu tidak valid: %s", timeStr)
	}
	return time.ParseInLocation(layout, timeStr[:19], loc)
}

func (s *Service) CheckAndAlert() {
	slog.Info("Cron job terpicu, memulai pengecekan Satnet...", "gateway", s.name)

	previousAlerts := s.state.GetActiveAlerts()
	degradedSatnets, err := s.getCurrentDownSatnets()
	if err != nil {
		slog.Error("Gagal mendapatkan status Satnet saat ini", "gateway", s.name, "error", err)
		return
	}

	if len(degradedSatnets) > 0 {
		slog.Info("Satnet terdeteksi DOWN, mengirim notifikasi...", "gateway", s.name, "count", len(degradedSatnets))
		report := types.GatewayReport{FriendlyName: s.name, Satnets: degradedSatnets}
		if err := s.notifier.SendSatnetAlert(report); err != nil {
			slog.Error("Gagal mengirim notifikasi Satnet DOWN", "gateway", s.name, "error", err)
		}
	}

	currentDownMap := make(map[string]types.SatnetDetail)
	for _, satnet := range degradedSatnets {
		currentDownMap[satnet.Name] = satnet
	}

	for _, satnetDetail := range degradedSatnets {
		alertKey := s.getAlertKey(satnetDetail.Name)
		if _, exists := previousAlerts[alertKey]; !exists {
			slog.Info("Menambahkan Satnet DOWN baru ke state", "gateway", s.name, "satnet", satnetDetail.Name)
			s.state.AddAlert(alertKey, state.ActiveAlert{
				Type:    "satnet",
				Gateway: s.name,
				Details: satnetDetail,
			})
		}
	}

	var recoveredSatnets []types.SatnetUpAlert
    prefix := fmt.Sprintf("satnet_%s_", s.name)
    for key, alert := range previousAlerts {
        if !strings.HasPrefix(key, prefix) {
            continue
        }
        satnetName := strings.TrimPrefix(key, prefix)
        if _, stillDown := currentDownMap[satnetName]; stillDown {
            continue
        }

        m, ok := alert.Details.(map[string]interface{})
        if !ok {
            slog.Warn("Details bukan map[string]interface{}", "key", key)
            continue
        }
        rawStart, _ := m["start_issue"].(string)
        tStart, err := ParseStringToWIB(rawStart)
        if err != nil {
            slog.Warn("Gagal parse start_issue, gunakan time.Now()", "raw", rawStart, "err", err)
            tStart = time.Now()
        }

        slog.Info("Satnet terdeteksi PULIH", "gateway", s.name, "satnet", satnetName)
        recoveredSatnets = append(recoveredSatnets, types.SatnetUpAlert{
            GatewayName:  s.name,
            SatnetName:   satnetName,
            RecoveryTime: time.Now(),
            TimeDown:     tStart,
        })

        s.state.RemoveAlertByKey(key)
    }
	if len(recoveredSatnets) > 0 {
		if err := s.notifier.SendSatnetUpAlert(recoveredSatnets); err != nil {
			slog.Error("Gagal mengirim notifikasi Satnet UP", "gateway", s.name, "error", err)
		}
	}
}

func (s *Service) getCurrentDownSatnets() ([]types.SatnetDetail, error) {
	const thresholdKbps = 1000.0
	const alertThreshold = 3

	allData, err := s.repo.GetLastSatnetData()
	if err != nil {
		return nil, err
	}

	var degradedSatnetsForReport []types.SatnetDetail
	for _, data := range allData {
		if data.FwdThroughput < thresholdKbps {
			online, offline, err := s.repo.GetTerminalStatus(data.Name)
			if err != nil {
				slog.Warn("Gagal mendapatkan status terminal", "gateway", s.name, "satnet", data.Name, "error", err)
			}

			var totalAffected int64
			if online != nil {
				totalAffected += *online
			}
			if offline != nil {
				totalAffected += *offline
			}

			if totalAffected > alertThreshold {
				startIssueTime, _ := s.repo.GetStartIssueTime(data.Name)
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
	return degradedSatnetsForReport, nil
}

func (s *Service) getAlertKey(satnetName string) string {
	return fmt.Sprintf("satnet_%s_%s", s.name, satnetName)
}
