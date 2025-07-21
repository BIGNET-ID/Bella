package db

import (
	configs "bella/config"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Connections struct {
	DBOneJYP  *gorm.DB
	DBOneMNK  *gorm.DB
	DBOneTMK  *gorm.DB
	DBFiveJYP *gorm.DB
	DBFiveMNK *gorm.DB
	DBFiveTMK *gorm.DB
}

func InitializeDatabases(config *configs.AppConfig) *Connections {
	conns := &Connections{}

	newLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	connect := func(cfg configs.DatabaseConfig, name string) *gorm.DB {
		if !cfg.IsConfigured {
			return nil
		}
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			cfg.Host, cfg.User, cfg.Pass, cfg.Name, cfg.Port)

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: newLogger,
		})
		if err != nil {
			log.Printf("PERINGATAN: Gagal konek ke database %s: %v. Melewati...", name, err)
			return nil
		}
		log.Printf("Koneksi ke database '%s' berhasil.", name)
		return db
	}

	conns.DBOneJYP = connect(config.DBOneJYP, "DB_ONE_JYP")
	conns.DBOneMNK = connect(config.DBOneMNK, "DB_ONE_MNK")
	conns.DBOneTMK = connect(config.DBOneTMK, "DB_ONE_TMK")
	conns.DBFiveJYP = connect(config.DBFiveJYP, "DB_FIVE_JYP")
	conns.DBFiveMNK = connect(config.DBFiveMNK, "DB_FIVE_MNK")
	conns.DBFiveTMK = connect(config.DBFiveTMK, "DB_FIVE_TMK")

	return conns
}
