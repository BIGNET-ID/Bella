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
	SendSatnetAlert(report types.GatewayReport) error
}

type telegramNotifier struct {
	botToken string
	chatID   string
}

func NewTelegramNotifier(token, chatID string) Notifier {
	return &telegramNotifier{botToken: token, chatID: chatID}
}

func (t *telegramNotifier) SendSatnetAlert(report types.GatewayReport) error {
	if len(report.Satnets) == 0 {
		return nil
	}

	var messageBuilder strings.Builder
	friendlyGatewayName := t.determineFriendlyGatewayName(report.FriendlyName)

	alertTitle := fmt.Sprintf("🚨 *CRITICAL ALERT: %d SATNETS DOWN* 🚨", len(report.Satnets))
	gatewayLine := fmt.Sprintf("🔴 *GATEWAY: %s*", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n%s\n%s\n\n", alertTitle, gatewayLine, escapeMarkdownV2("────────────────────────────────"))
	messageBuilder.WriteString(header)

	var latestTime time.Time

	for _, satnet := range report.Satnets {
		onlineStr := "\\-"
		if satnet.OnlineCount != nil {
			onlineStr = fmt.Sprintf("%d", *satnet.OnlineCount)
		}

		offlineStr := "\\-"
		if satnet.OfflineCount != nil {
			offlineStr = fmt.Sprintf("%d", *satnet.OfflineCount)
		}

		fwdStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.FwdTp))
		rtnStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.RtnTp))

		satnetInfo := fmt.Sprintf(
			"🔻 *SATNET:* %s\n"+
				"   ├─ *Fwd:* %s kbps `(LOW)`\n"+
				"   ├─ *Rtn:* %s kbps\n"+
				"   ├─ *Online:* %s\n"+
				"   └─ *Offline:* %s\n\n",
			escapeMarkdownV2(satnet.Name),
			fwdStr,
			rtnStr,
			onlineStr,
			offlineStr,
		)
		messageBuilder.WriteString(satnetInfo)

		parsedTime, err := time.Parse(time.RFC3339, satnet.Time)
		if err == nil && parsedTime.After(latestTime) {
			latestTime = parsedTime
		}
	}

	detectionTimeStr := "N/A"
	if !latestTime.IsZero() {
		detectionTimeStr = latestTime.Format("2006-01-02 15:04:05 WIB")
	}


	tagLine := "👥 *CC:* @burhanudinus @mardamar99 @DelMelo @sepatubapak \\(mohon perhatiannya\\)"
	footer := fmt.Sprintf("🕒 *Time of Detection:* %s\n\n%s\n\n*ACTION:* Immediate investigation required\\.",
		escapeMarkdownV2(detectionTimeStr),
		tagLine,
	)
	messageBuilder.WriteString(footer)

	return t.sendMessage(messageBuilder.String())
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
	replacer := strings.NewReplacer("_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-", "=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!")
	return replacer.Replace(text)
}

func (t *telegramNotifier) determineFriendlyGatewayName(gatewayName string) string {
	upperName := strings.ToUpper(gatewayName)
	if strings.Contains(upperName, "JYP") {
		return "JAYAPURA"
	}
	if strings.Contains(upperName, "MNK") {
		return "MANOKWARI"
	}
	if strings.Contains(upperName, "TMK") {
		return "TIMIKA"
	}
	return gatewayName
}
