package setup

import (
	config "bella/config"
	"bella/db"
	"bella/internal/notifier"
	"bella/internal/satnet"
	"log"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

var dbFiveMap = map[string]*gorm.DB{}

func RegisterServicesAndTasks(
	allConnections *db.Connections,
	notifier notifier.Notifier,
	scheduler *cron.Cron,
	config *config.AppConfig,
) {
	log.Println("Mendaftarkan service dan tugas terjadwal...")

	dbFiveMap["DB_ONE_JYP"] = allConnections.DBFiveJYP
	dbFiveMap["DB_ONE_MNK"] = allConnections.DBFiveMNK
	dbFiveMap["DB_ONE_TMK"] = allConnections.DBFiveTMK

	register := func(dbOneName string, dbOneConn *gorm.DB) {
		if dbOneConn == nil {
			log.Printf("Koneksi untuk %s tidak aktif, tugas dilewati.", dbOneName)
			return
		}

		dbFiveConn, ok := dbFiveMap[dbOneName]
		if !ok || dbFiveConn == nil {
			log.Printf("PERINGATAN: Koneksi DB_FIVE untuk %s tidak ditemukan/aktif. Tugas tidak akan menyertakan status terminal.", dbOneName)
			return
		}

		satnetSvc := satnet.NewService(dbFiveConn, notifier, dbOneName)
		scheduler.AddFunc(config.CronSchedule, satnetSvc.CheckAndAlert)
		log.Printf("Service Satnet (dengan status terminal) untuk '%s' berhasil didaftarkan.", dbOneName)
	}

	register("DB_ONE_JYP", allConnections.DBOneJYP)
	register("DB_ONE_MNK", allConnections.DBOneMNK)
	register("DB_ONE_TMK", allConnections.DBOneTMK)
}
