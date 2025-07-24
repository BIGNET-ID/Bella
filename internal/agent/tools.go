package agent

import (
	"bella/db"
	"bella/internal/notifier"
	"context"
	"fmt"
	"time"

	"github.com/pontus-devoteam/agent-sdk-go/pkg/tool"
	"gorm.io/gorm"
)

type Toolset struct {
	dbConnections *db.Connections
	notifier      notifier.Notifier
}

func NewToolset(conns *db.Connections, notifier notifier.Notifier) *Toolset {
	return &Toolset{dbConnections: conns, notifier: notifier}
}

func (ts *Toolset) GetDegradedSatnetsTool() *tool.FunctionTool {
	return tool.NewFunctionTool(
		"get_degraded_satnets",
		"Mendapatkan daftar semua satnet yang performanya di bawah ambang batas (1000 kbps) dari sebuah gateway DB_ONE.",
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			gatewayName, ok := params["gateway_name"].(string)
			if !ok {
				return nil, fmt.Errorf("argumen 'gateway_name' tidak valid atau tidak ada")
			}

			var db *gorm.DB
			switch gatewayName {
			case "DB_ONE_JYP":
				db = ts.dbConnections.DBOneJYP
			case "DB_ONE_MNK":
				db = ts.dbConnections.DBOneMNK
			case "DB_ONE_TMK":
				db = ts.dbConnections.DBOneTMK
			default:
				return nil, fmt.Errorf("gateway %s tidak dikenal atau bukan bagian dari DB_ONE", gatewayName)
			}

			if db == nil {
				return nil, fmt.Errorf("database untuk %s tidak aktif", gatewayName)
			}

			type SatnetResult struct {
				Name  string    `json:"name"`
				FwdTp float64   `json:"fwd_tp"`
				RtnTp float64   `json:"rtn_tp"`
				Time  time.Time `json:"time"`
			}
			var results []SatnetResult
			err := db.Raw(`
				SELECT DISTINCT ON (satnet_name)
					satnet_name as name,
					satnet_fwd_throughput as fwd_tp,
					satnet_rtn_throughput as rtn_tp,
					updated_at as time
				FROM satnets
				ORDER BY satnet_name, updated_at DESC
			`).Scan(&results).Error
			if err != nil {
				return nil, err
			}

			var degradedResults []SatnetResult
			for _, r := range results {
				if r.FwdTp < 1000.0 {
					degradedResults = append(degradedResults, r)
				}
			}

			return degradedResults, nil
		},
	).WithSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"gateway_name": map[string]interface{}{
				"type": "string", "description": "Nama gateway dari grup DB_ONE, contoh: DB_ONE_JYP",
			},
		},
		"required": []string{"gateway_name"},
	})
}

func (ts *Toolset) GetTerminalStatusTool() *tool.FunctionTool {
	return tool.NewFunctionTool(
		"get_terminal_status",
		"Untuk satu satnet spesifik, dapatkan jumlah terminal yang online (avg_esno > 0) dan offline dari data terbaru di gateway DB_FIVE yang sesuai.",
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			gatewayName, _ := params["gateway_name"].(string)
			satnetName, _ := params["satnet_name"].(string)

			var db5 *gorm.DB
			switch gatewayName {
			case "DB_ONE_JYP":
				db5 = ts.dbConnections.DBFiveJYP
			case "DB_ONE_MNK":
				db5 = ts.dbConnections.DBFiveMNK
			case "DB_ONE_TMK":
				db5 = ts.dbConnections.DBFiveTMK
			default:
				return nil, fmt.Errorf("gateway %s tidak memiliki padanan di DB_FIVE", gatewayName)
			}

			if db5 == nil {
				return nil, fmt.Errorf("database DB_FIVE untuk %s tidak aktif", gatewayName)
			}

			var recordCount int64
			if err := db5.Table("modem_kpi").Where("satnet = ?", satnetName).Count(&recordCount).Error; err != nil {
				return nil, fmt.Errorf("gagal melakukan pre-check count untuk satnet %s: %w", satnetName, err)
			}

			if recordCount == 0 {
				return map[string]int64{"online_count": -1, "offline_count": -1}, nil
			}

			var latestTime struct {
				Time time.Time
			}
			err := db5.Table("modem_kpi").Select("MAX(time) as time").Where("satnet = ?", satnetName).Scan(&latestTime).Error
			if err != nil {
				return nil, fmt.Errorf("gagal menemukan data waktu terakhir untuk satnet %s: %w", satnetName, err)
			}

			type StatusResult struct {
				OnlineCount  int64 `json:"online_count"`
				OfflineCount int64 `json:"offline_count"`
			}
			var result StatusResult

			db5.Table("modem_kpi").Where("satnet = ? AND time = ? AND esno_avg > 0", satnetName, latestTime.Time).Count(&result.OnlineCount)
			db5.Table("modem_kpi").Where("satnet = ? AND time = ? AND (esno_avg <= 0 OR esno_avg IS NULL)", satnetName, latestTime.Time).Count(&result.OfflineCount)

			return result, nil
		},
	).WithSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"gateway_name": map[string]interface{}{"type": "string", "description": "Gateway asal dari grup DB_ONE, contoh: DB_ONE_JYP."},
			"satnet_name":  map[string]interface{}{"type": "string", "description": "Nama satnet yang akan dicek, contoh: JYPN1-B001-SN01."},
		},
		"required": []string{"gateway_name", "satnet_name"},
	})
}
