package notifier

import (
	"bella/internal/types"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Notifier interface {
	FormatAndSendAgentReport(report types.GatewayReport) error
}

type telegramNotifier struct {
	botToken string
	chatID   string
}

func NewTelegramNotifier(token, chatID string) Notifier {
	return &telegramNotifier{botToken: token, chatID: chatID}
}

func (t *telegramNotifier) FormatAndSendAgentReport(report types.GatewayReport) error {
	log.Printf("✅ [NOTIFIER] Memulai format laporan untuk Gateway: %s", report.FriendlyName)

	if len(report.Satnets) == 0 {
		log.Printf("🟡 [NOTIFIER] Tidak ada satnet bermasalah untuk dilaporkan di %s. Melewati.", report.FriendlyName)
		return nil
	}

	var finalReport strings.Builder

	friendlyName := t.determineFriendlyGatewayName(report.FriendlyName)

	finalReport.WriteString(fmt.Sprintf("🚨 *CRITICAL ALERT: %d SATNETS DOWN* 🚨\n", len(report.Satnets)))
	finalReport.WriteString(fmt.Sprintf("🔰 *GATEWAY: %s*\n", escapeMarkdownV2(friendlyName)))
	finalReport.WriteString(escapeMarkdownV2("────────────────────────────────") + "\n\n")

	for _, satnet := range report.Satnets {
		var onlineStr, offlineStr string

		if satnet.OnlineCount == nil || *satnet.OnlineCount == -1 {
			onlineStr = "\\-"
		} else {
			onlineStr = fmt.Sprintf("%d", *satnet.OnlineCount)
		}

		if satnet.OfflineCount == nil || *satnet.OfflineCount == -1 {
			offlineStr = "\\-"
		} else {
			offlineStr = fmt.Sprintf("%d", *satnet.OfflineCount)
		}

		fwdStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.FwdTp))
		rtnStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.RtnTp))
		satnetNameStr := escapeMarkdownV2(satnet.Name)

		finalReport.WriteString(fmt.Sprintf("🔻 *SATNET:* %s\n", satnetNameStr))
		finalReport.WriteString(fmt.Sprintf("   ├─ *Fwd:* %s kbps `(LOW)`\n", fwdStr))
		finalReport.WriteString(fmt.Sprintf("   ├─ *Rtn:* %s kbps\n", rtnStr))
		finalReport.WriteString(fmt.Sprintf("   ├─ *Online:* %s\n", onlineStr))
		finalReport.WriteString(fmt.Sprintf("   └─ *Offline:* %s\n\n", offlineStr))
	}

	detectionTime := time.Now().Format("2006-01-02 15:04:05 MST")
	if len(report.Satnets) > 0 && report.Satnets[0].Time != "" {
		parsedTime, err := time.Parse(time.RFC3339, report.Satnets[0].Time)
		if err == nil {
			detectionTime = parsedTime.Format("2006-01-02 15:04:05 MST")
		}
	}

	tagLine := "👥 *CC:* @x @x \\(mohon perhatiannya\\)"
	footer := fmt.Sprintf("🕒 *Time of Detection:* %s\n\n%s\n\n*ACTION:* Immediate investigation required\\.",
		escapeMarkdownV2(detectionTime),
		tagLine,
	)
	finalReport.WriteString(footer)

	log.Printf("📤 [NOTIFIER] Laporan untuk %s sudah diformat, mencoba mengirim...", report.FriendlyName)
	return t.sendMessage(finalReport.String())
}

func (t *telegramNotifier) sendMessage(text string) error {
	payload := map[string]string{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "MarkdownV2",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body bytes.Buffer
		body.ReadFrom(resp.Body)
		log.Printf("❌ [NOTIFIER] Gagal mengirim ke Telegram! Status: %d, Pesan: %s", resp.StatusCode, body.String())
		return fmt.Errorf("telegram API Error: %s (status: %d)", body.String(), resp.StatusCode)
	}

	log.Println("✅ [NOTIFIER] Pesan berhasil dikirim ke Telegram.")
	return nil
}

func escapeMarkdownV2(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
		"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
	)
	return replacer.Replace(text)
}

func (t *telegramNotifier) determineFriendlyGatewayName(gatewayName string) string {
	if strings.Contains(gatewayName, "JYP") {
		return "JAYAPURA"
	}
	if strings.Contains(gatewayName, "MNK") {
		return "MANOKWARI"
	}
	if strings.Contains(gatewayName, "TMK") {
		return "TIMIKA"
	}
	return gatewayName
}
