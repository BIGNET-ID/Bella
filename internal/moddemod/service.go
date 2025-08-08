package moddemod

import (
	"bella/internal/notifier"
	"bella/internal/state"
	"bella/internal/types"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	repo     Repository
	notifier notifier.Notifier
	state    *state.Manager
	name     string
}

func NewService(dbOne *gorm.DB, notifier notifier.Notifier, stateMgr *state.Manager, name string) *Service {
	return &Service{
		repo:     NewGormRepository(dbOne),
		notifier: notifier,
		state:    stateMgr,
		name:     name,
	}
}

func ParseWIBTimestamp(timeStr string) (time.Time, error) {
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
	slog.Info("Cron job terpicu, memulai pengecekan Modulator/Demodulator...", "gateway", s.name)
	s.checkDevices("modulator")
	s.checkDevices("demodulator")
}

func (s *Service) checkDevices(deviceType string) {
	var currentDownDevices []DeviceStatus
	var err error

	if deviceType == "modulator" {
		currentDownDevices, err = s.repo.GetDownModulators()
	} else {
		currentDownDevices, err = s.repo.GetDownDemodulators()
	}
	if err != nil {
		slog.Error("Gagal mendapatkan status perangkat", "type", deviceType, "gateway", s.name, "error", err)
		return
	}

	previousAlerts := s.state.GetActiveAlerts()

	if len(currentDownDevices) > 0 {
		slog.Info("Perangkat terdeteksi DOWN, mengirim notifikasi...", "gateway", s.name, "type", deviceType, "count", len(currentDownDevices))
		downAlerts := []types.ModemDownAlert{}
		for _, deviceStatus := range currentDownDevices {
			downAlerts = append(downAlerts, types.ModemDownAlert{
				GatewayName: s.name,
				DeviceName:  deviceStatus.DeviceName,
				AlarmState:  deviceStatus.AlarmState,
				StartTime:   deviceStatus.UpdatedAt,
			})
		}
		if err := s.notifier.SendModemDownAlert(downAlerts, deviceType); err != nil {
			slog.Error("Gagal mengirim notifikasi DOWN", "gateway", s.name, "type", deviceType, "error", err)
		}
	}

	currentDownMap := make(map[string]DeviceStatus)
	for _, dev := range currentDownDevices {
		currentDownMap[dev.DeviceName] = dev
	}

	for _, deviceStatus := range currentDownDevices {
		alertKey := s.getAlertKey(deviceStatus.DeviceName, deviceType)
		if _, exists := previousAlerts[alertKey]; !exists {
			slog.Info("Menambahkan perangkat DOWN baru ke state", "gateway", s.name, "type", deviceType, "device", deviceStatus.DeviceName)
			s.state.AddAlert(alertKey, state.ActiveAlert{
				Type:    deviceType,
				Gateway: s.name,
				Details: deviceStatus,
			})
		}
	}

	recoveredAlerts := []types.ModemUpAlert{}
	prefix := fmt.Sprintf("%s_%s_", deviceType, s.name)

	for key, alertData := range previousAlerts {
		if strings.HasPrefix(key, prefix) {
			deviceName := strings.TrimPrefix(key, prefix)
			if _, stillDown := currentDownMap[deviceName]; !stillDown {

				var downTime time.Time

				if detailsMap, ok := alertData.Details.(map[string]interface{}); ok {
					if updatedAtStr, ok := detailsMap["UpdatedAt"].(string); ok {

						parsedTime, err := ParseWIBTimestamp(updatedAtStr)
						if err != nil {
							slog.Error("Gagal parsing 'UpdatedAt' untuk modem pulih", "key", key, "error", err)
							downTime = time.Now()
						} else {
							downTime = parsedTime
						}
					} else {
						slog.Warn("Field 'UpdatedAt' tidak ditemukan atau bukan string", "key", key)
						downTime = time.Now()
					}
				} else {
					slog.Warn("Format 'Details' tidak terduga untuk modem pulih", "key", key)
					downTime = time.Now()
				}

				slog.Info("Perangkat terdeteksi PULIH", "gateway", s.name, "type", deviceType, "device", deviceName)

				recoveredAlerts = append(recoveredAlerts, types.ModemUpAlert{
					GatewayName:  s.name,
					DeviceName:   deviceName,
					RecoveryTime: time.Now(),
					TimeDown:     downTime,
				})
				s.state.RemoveAlertByKey(key)
			}
		}
	}

	if len(recoveredAlerts) > 0 {
		if err := s.notifier.SendModemUpAlert(recoveredAlerts, deviceType); err != nil {
			slog.Error("Gagal mengirim notifikasi UP", "gateway", s.name, "type", deviceType, "error", err)
		}
	}
}

func (s *Service) getAlertKey(deviceName, deviceType string) string {
	return fmt.Sprintf("%s_%s_%s", deviceType, s.name, deviceName)
}
