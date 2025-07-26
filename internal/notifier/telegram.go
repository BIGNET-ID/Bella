package notifier

import (
	"bella/internal/types"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

	alertTitle := "🚨 *CRITICAL ALERT* 🚨"
	eventLine := fmt.Sprintf("🔴 *EVENT:* %d SATNETS DOWN", len(report.Satnets))
	gatewayLine := fmt.Sprintf("🔰 *GATEWAY:* %s", escapeMarkdownV2(friendlyGatewayName))
	header := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n",
		alertTitle,
		eventLine,
		gatewayLine,
		escapeMarkdownV2("────────────────────────────────"),
	)
	messageBuilder.WriteString(header)

	for _, satnet := range report.Satnets {
		onlineStr := "0"
		if satnet.OnlineCount != nil {
			onlineStr = fmt.Sprintf("%d", *satnet.OnlineCount)
		}

		offlineStr := "0"
		if satnet.OfflineCount != nil {
			offlineStr = fmt.Sprintf("%d", *satnet.OfflineCount)
		}

		startIssueStr := "N/A"
		if satnet.StartIssue != nil {
			startIssueStr = escapeMarkdownV2(satnet.StartIssue.Format("2006-01-02 15:04:05 WIB"))
		}

		fwdStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.FwdTp))
		rtnStr := escapeMarkdownV2(fmt.Sprintf("%.2f", satnet.RtnTp))

		satnetInfo := fmt.Sprintf(
			"🔻 *SATNET:* %s\n"+
				"   ├─ *FWD :* %s kbps `(LOW)`\n"+
				"   ├─ *RTN :* %s kbps\n"+
				"   ├─ *Online UT :* %s\n"+
				"   ├─ *Offline UT :* %s\n"+
				"   └─ *Start :* %s\n\n",
			escapeMarkdownV2(satnet.Name),
			fwdStr,
			rtnStr,
			onlineStr,
			offlineStr,
			startIssueStr,
		)
		messageBuilder.WriteString(satnetInfo)
	}

	footer := "*ACTION:* Immediate investigation required\\."
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
