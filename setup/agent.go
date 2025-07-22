package setup

import (
	config "bella/config"
	"bella/db"
	"bella/internal/agent"
	"bella/internal/notifier"
	"bella/internal/types"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	sdk_agent "github.com/pontus-devoteam/agent-sdk-go/pkg/agent"
	"github.com/pontus-devoteam/agent-sdk-go/pkg/model/providers/openai"
	"github.com/pontus-devoteam/agent-sdk-go/pkg/runner"
	"github.com/robfig/cron/v3"
)

// RegisterAgentTasks mendaftarkan tugas untuk menjalankan Agent secara periodik.
func RegisterAgentTasks(
	allConnections *db.Connections,
	notifier notifier.Notifier,
	scheduler *cron.Cron,
	config *config.AppConfig,
) {
	log.Println("Mendaftarkan tugas untuk Agent...")

	// 1. Buat Provider untuk LLM (OpenAI)
	provider := openai.NewProvider(config.OpenAIApiKey)
	provider.SetDefaultModel("gpt-4.1-nano")

	// 2. Buat Toolset dengan dependensi yang dibutuhkan
	toolset := agent.NewToolset(allConnections, notifier)

	// 3. Buat Agent dari library SDK
	monitoringAgent := sdk_agent.NewAgent("MonitoringAssistant")
	monitoringAgent.SetModelProvider(provider)
	monitoringAgent.WithModel("gpt-4.1-nano")
	// Instruksi yang jauh lebih singkat dan efisien, dengan contoh JSON
	monitoringAgent.SetSystemInstructions(`You are a network monitoring assistant. For a given gateway, find all degraded satnets and get the terminal status for each of them. You must call the tools to get the data. Finally, you must return all the collected information for that single gateway in a final JSON object that strictly follows this exact structure: {"friendly_name": "...", "satnets": [{"name": "...", "fwd_tp": ..., "rtn_tp": ..., "time": "...", "online_count": ..., "offline_count": ...}]}. Do not add any conversational text.`)

	// Tambahkan semua tools yang tersedia ke Agent
	monitoringAgent.WithTools(
		toolset.GetDegradedSatnetsTool(),
		toolset.GetTerminalStatusTool(),
	)

	// 4. Buat Runner
	agentRunner := runner.NewRunner()
	agentRunner.WithDefaultProvider(provider)

	// 5. Jadwalkan tugas cron untuk menjalankan Agent
	scheduler.AddFunc(config.CronSchedule, func() {
		log.Println("⏰ Cron terpicu, menjalankan Agent untuk semua gateway DB_ONE...")

		gatewaysToCheck := []string{"DB_ONE_JYP", "DB_ONE_MNK", "DB_ONE_TMK"}
		for _, gwName := range gatewaysToCheck {
			// Jalankan agent untuk setiap gateway secara terpisah di goroutine
			go func(gateway string) {
				log.Printf("▶️ Memulai agent untuk gateway: %s", gateway)

				input := fmt.Sprintf("Please check the network status for gateway %s now and return the structured JSON report.", gateway)

				result, err := agentRunner.RunSync(monitoringAgent, &runner.RunOptions{
					Input: input,
				})
				if err != nil {
					log.Printf("❌ Error saat menjalankan agent untuk %s: %v", gateway, err)
					return
				}

				if result.FinalOutput == nil || result.FinalOutput == "" {
					log.Printf("✅ Agent untuk %s selesai. Tidak ada yang perlu dilaporkan.", gateway)
					return
				}

				finalOutputStr, ok := result.FinalOutput.(string)
				if !ok {
					log.Printf("❌ Gagal mengonversi output agent untuk %s ke string.", gateway)
					return
				}

				log.Printf("📝 Agent untuk %s selesai. Hasil JSON: %s", gateway, finalOutputStr)

				var report types.GatewayReport
				jsonString := strings.Trim(finalOutputStr, " \n\t`")

				err = json.Unmarshal([]byte(jsonString), &report)
				if err != nil {
					log.Printf("❌ Gagal mem-parsing JSON dari Agent untuk %s. Error: %v", gateway, err)
					return // Hentikan jika parsing gagal
				}
				log.Printf("✅ [SETUP] JSON untuk %s berhasil di-parse.", gateway)

				// Kirim laporan untuk gateway ini ke notifier
				err = notifier.FormatAndSendAgentReport(report)
				if err != nil {
					log.Printf("❌ Gagal mengirim laporan untuk %s: %v", gateway, err)
				}
			}(gwName)
		}
	})

	log.Printf("Agent berhasil dijadwalkan untuk berjalan setiap: %s", config.CronSchedule)
}
