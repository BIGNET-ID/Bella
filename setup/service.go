package setup

import (
	config "bella/config"
	"bella/db"
	"bella/internal/moddemod"
	"bella/internal/notifier"
	"bella/internal/prtgn"
	"bella/internal/satnet"
	"bella/internal/state"
	"log/slog"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

func RegisterServices(allConnections *db.Connections, notifier notifier.Notifier, stateMgr *state.Manager) map[string]*satnet.Service {
	slog.Info("Menginisialisasi semua service...")
	serviceMap := make(map[string]*satnet.Service)

	dbFiveMap := map[string]*gorm.DB{
		"JAYAPURA":  allConnections.DBFiveJYP,
		"MANOKWARI": allConnections.DBFiveMNK,
		"TIMIKA":    allConnections.DBFiveTMK,
	}

	for name, dbConn := range dbFiveMap {
		if dbConn != nil {
			serviceMap[name] = satnet.NewService(dbConn, notifier, stateMgr, name)
			slog.Info("Service Satnet untuk gateway berhasil dibuat.", "gateway", name)
		}
	}
	return serviceMap
}

func RegisterCronJobs(scheduler *cron.Cron, config *config.AppConfig, serviceMap map[string]*satnet.Service, prtgAPI prtgn.PRTGAPIInterface, allConnections *db.Connections, notifier notifier.Notifier, stateMgr *state.Manager) {
	slog.Info("Mendaftarkan tugas-tugas cron...")

	for name, service := range serviceMap {
		svc := service
		scheduler.AddFunc(config.CronSchedule, svc.CheckAndAlert)
		slog.Info("Tugas cron Satnet berhasil didaftarkan.", "gateway", name)
	}

	if prtgAPI != nil {
		scheduler.AddFunc(config.CronSchedule, prtgAPI.RunPeriodicChecks)
		slog.Info("Tugas cron untuk Pengecekan PRTG (NIF & IPTX) berhasil didaftarkan.")
	}

	dbOneMap := map[string]*gorm.DB{
		"JAYAPURA":  allConnections.DBOneJYP,
		"MANOKWARI": allConnections.DBOneMNK,
		"TIMIKA":    allConnections.DBOneTMK,
	}

	for name, dbConn := range dbOneMap {
		if dbConn != nil {
			modemService := moddemod.NewService(dbConn, notifier, stateMgr, name)
			scheduler.AddFunc(config.CronSchedule, modemService.CheckAndAlert)
			slog.Info("Tugas cron Modulator/Demodulator berhasil didaftarkan.", "gateway", name)
		}
	}
}
