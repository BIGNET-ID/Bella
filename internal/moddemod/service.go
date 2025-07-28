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
	currentDownMap := make(map[string]DeviceStatus)
	for _, dev := range currentDownDevices {
		currentDownMap[dev.DeviceName] = dev
	}

	newDownAlerts := []types.ModemDownAlert{}
	for deviceName, deviceStatus := range currentDownMap {
		alertKey := s.getAlertKey(deviceName, deviceType)
		if _, exists := previousAlerts[alertKey]; !exists {
			slog.Info("Perangkat BARU terdeteksi DOWN", "gateway", s.name, "type", deviceType, "device", deviceName)
			newDownAlerts = append(newDownAlerts, types.ModemDownAlert{
				GatewayName: s.name,
				DeviceName:  deviceName,
				AlarmState:  deviceStatus.AlarmState,
				StartTime:   deviceStatus.UpdatedAt,
			})
			s.state.AddAlert(alertKey, state.ActiveAlert{
				Type:    deviceType,
				Gateway: s.name,
				Details: deviceStatus,
			})
		}
	}
	if len(newDownAlerts) > 0 {
		if err := s.notifier.SendModemDownAlert(newDownAlerts, deviceType); err != nil {
			slog.Error("Gagal mengirim notifikasi DOWN", "gateway", s.name, "type", deviceType, "error", err)
		}
	}

	recoveredAlerts := []types.ModemUpAlert{}
	prefix := fmt.Sprintf("%s_%s_", deviceType, s.name)
	for key := range previousAlerts {
		if strings.HasPrefix(key, prefix) {
			deviceName := strings.TrimPrefix(key, prefix)
			if _, stillDown := currentDownMap[deviceName]; !stillDown {
				slog.Info("Perangkat terdeteksi PULIH", "gateway", s.name, "type", deviceType, "device", deviceName)
				recoveredAlerts = append(recoveredAlerts, types.ModemUpAlert{
					GatewayName:  s.name,
					DeviceName:   deviceName,
					RecoveryTime: time.Now(),
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
