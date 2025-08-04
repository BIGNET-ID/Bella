package bot

import (
	"bella/api"
	"fmt"
	"strings"
	"time"
)

// GetHelpMessage membuat pesan bantuan yang dinamis berdasarkan status otorisasi pengguna.
func GetHelpMessage(isAdmin bool) string {
	var sb strings.Builder

	sb.WriteString("👋 *Selamat Datang di Bella Bot Monitoring* 👋\n\n")
	sb.WriteString("Saya adalah asisten virtual untuk memantau kondisi jaringan SATRIA\\-1\\. Berikut adalah daftar perintah yang bisa Anda gunakan:\n\n")

	if isAdmin {
		// Tampilan untuk Admin
		sb.WriteString("🛰️ *Perintah Status Gateway*\n")
		sb.WriteString(escape("───────────────\n"))
		sb.WriteString("`/satria1_gateway_all` \\- Ringkasan status semua Gateway\n")
		sb.WriteString("`/satria1_gateway_jyp` \\- Ringkasan status Gateway Jayapura\n")
		sb.WriteString("`/satria1_gateway_mnk` \\- Ringkasan status Gateway Manokwari\n")
		sb.WriteString("`/satria1_gateway_tmk` \\- Ringkasan status Gateway Timika\n\n")

		sb.WriteString("📊 *Perintah Info IP Transit*\n")
		sb.WriteString(escape("───────────────\n"))
		sb.WriteString("`/satria1_iptx_jyp` \\- Info IP Transit Gateway Jayapura\n")
		sb.WriteString("`/satria1_iptx_mnk` \\- Info IP Transit Gateway Manokwari\n")
		sb.WriteString("`/satria1_iptx_tmk` \\- Info IP Transit Gateway Timika\n\n")

		sb.WriteString("🛠️ *Perintah Log & Diagnostik*\n")
		sb.WriteString(escape("───────────────\n"))
		sb.WriteString("`/log_error` \\- Tampilkan 15 log error terakhir\n")
		sb.WriteString("`/log_notif` \\- Tampilkan 15 log notifikasi terakhir\n")
		sb.WriteString("`/log_alerts_active` \\- Tampilkan semua alert yang aktif\n")
		sb.WriteString("`/log_all` \\- Tampilkan 20 log mentah terakhir\n\n")

		sb.WriteString("⚙️ *Perintah Umum*\n")
		sb.WriteString(escape("───────────────\n"))
		sb.WriteString("`/myid` \\- Menampilkan ID Telegram Anda\n")
		sb.WriteString("`/help` \\- Menampilkan pesan bantuan ini\n\n")

		sb.WriteString("💡 *Tips:* Anda memiliki akses penuh sebagai *Admin*\\.\n")

	} else {
		// Tampilan untuk Pengguna Publik
		sb.WriteString("🔰 *Perintah*\n")
		sb.WriteString(escape("───────────────\n"))
		sb.WriteString("`/start` \\- Memulai interaksi dengan bot\n")
		sb.WriteString("`/help` \\- Menampilkan pesan bantuan ini\n")
		sb.WriteString("`/myid` \\- Menampilkan ID Telegram Anda\n\n")

		sb.WriteString("🔒 *Akses Terbatas*\n")
		sb.WriteString("Anda menggunakan bot sebagai pengguna publik\\. Untuk mendapatkan akses ke fitur admin \\(seperti melihat status gateway dan log\\), silakan berikan ID Telegram Anda kepada administrator sistem\\.\n")
	}

	return sb.String()
}


// FormatMyIDMessage memformat pesan untuk perintah /myid.
func FormatMyIDMessage(userID int64) string {
	return fmt.Sprintf("ID Telegram Anda adalah: `%d`", userID)
}

// FormatLogMessage memformat output log agar mudah dibaca dalam blok kode.
func FormatLogMessage(title, content string) string {
	if strings.TrimSpace(content) == "" {
		content = "Tidak ada data untuk ditampilkan."
	}
	// Menggunakan triple backtick untuk blok kode dan escape judul
	return fmt.Sprintf("*%s*\n```\n%s\n```", escape(title), content)
}


func escape(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
		"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
	)
	return replacer.Replace(text)
}

func statusToEmoji(status string) string {
	switch strings.ToLower(status) {
	case "up":
		return "🟩"
	case "down":
		return "🟥"
	default:
		return status
	}
}

func FormatGatewayHeader(gatewayName string) string {
	var b strings.Builder
	now := time.Now().Format("02 Jan 2006 15:04:05 WIB")
	b.WriteString(fmt.Sprintf("📡 *Gateway %s Status Report*\n", escape(gatewayName)))
	b.WriteString(fmt.Sprintf("`      (%s)`\n", escape(now)))
	return b.String()
}

func formatSystemStatus(data GatewayData) string {
	var b strings.Builder
	b.WriteString("\n🔧 *System Status*\n")
	if data.IpcnStatus != nil {
		mainStatus := escape(data.IpcnStatus.IpTransitMain.StatusText)
		backupStatus := escape(data.IpcnStatus.IpTransitBackupStatus.StatusText)
		nifStatus := escape(data.IpcnStatus.NifStatus.StatusText)
		nmsStatus := escape(data.IpcnStatus.NmsStatus.StatusText)

		b.WriteString(fmt.Sprintf("`     ┌─ IP Transit Status : Main %s || Backup %s`\n", escape(statusToEmoji(mainStatus)), escape(statusToEmoji(backupStatus))))
		b.WriteString(fmt.Sprintf("`     ├─ Dataplane Status  : %s`\n", escape(statusToEmoji(nifStatus))))
		b.WriteString(fmt.Sprintf("`     └─ NMS Status        : %s`\n", escape(statusToEmoji(nmsStatus))))
	} else {
		b.WriteString(escape("     - Gagal mengambil data\n"))
	}
	return b.String()
}

func formatTrafficInfo(data GatewayData) string {
	var b strings.Builder
	b.WriteString("\n📊 *Traffic Info*\n")
	if data.IptxTraffic != nil && len(data.IptxTraffic.HisData) > 0 {
		trafficStr := fmt.Sprintf("%.2f", data.IptxTraffic.HisData[0].TrafficTotalSpeed)
		b.WriteString(fmt.Sprintf("`     ┌─  IPTX Aggregate Traffic : %s Mbps`\n", escape(trafficStr)))
	} else {
		b.WriteString(escape("     ┌─  IPTX Aggregate Traffic : Gagal mengambil data") + "\n")
	}

	if data.OnlineUT != nil && len(data.OnlineUT.Data) > 0 {
		latestUT := data.OnlineUT.Data[len(data.OnlineUT.Data)-1]
		b.WriteString(fmt.Sprintf("`     └─  Online UT              : %d`\n", latestUT.UtOnlineToa))
	} else {
		b.WriteString(escape("     └─  Online UT              : Gagal mengambil data") + "\n")
	}
	return b.String()
}

func formatIpcnDeviceDetails(data GatewayData, gatewayName string) string {
	var b strings.Builder
	b.WriteString("\n⚙️ *IPCN Device Status*\n")
	if data.IpcnSensors == nil {
		b.WriteString(escape("- Gagal mengambil data\n"))
		return b.String()
	}

	categories := categorizeSensors(data.IpcnSensors, gatewayName)
	order := []string{"Core Router", "Core Switch", "Management Router", "Management Switch", "Firewall", "VPN Gateway", "CHR Mikrotik", "Sandvine", "Server"}

	hasContent := false
	for _, name := range order {
		category, ok := categories[name]
		if !ok || category == nil || len(category.Devices) == 0 {
			continue
		}
		hasContent = true
		b.WriteString(fmt.Sprintf("*%s:*\n", escape(category.Name)))
		for i, device := range category.Devices {
			connector := "├"
			if i == len(category.Devices)-1 {
				connector = "└"
			}
			b.WriteString(fmt.Sprintf("`     %s─ %s : %s`\n", connector, escape(device.DeviceName), escape(statusToEmoji(device.StatustextPing))))
		}
	}

	if !hasContent {
		b.WriteString(escape("     - Tidak ada perangkat IPCN yang terdeteksi untuk gateway ini.\n"))
	}

	return b.String()
}

func formatIpcnDeviceSummary(data GatewayData, gatewayName string) string {
	var b strings.Builder
	b.WriteString("\n⚙️ *IPCN Device Status*\n")
	if data.IpcnSensors == nil {
		b.WriteString(escape("- Gagal mengambil data\n"))
		return b.String()
	}

	categories := categorizeSensors(data.IpcnSensors, gatewayName)
	order := []string{"Core Router", "Core Switch", "Management Router", "Management Switch", "Firewall", "VPN Gateway", "CHR Mikrotik", "Sandvine", "Server"}

	hasContent := false
	for _, name := range order {
		category, ok := categories[name]
		if !ok || category == nil || len(category.Devices) == 0 {
			continue
		}
		hasContent = true
		up, down := 0, 0
		for _, device := range category.Devices {
			if strings.ToLower(device.StatustextPing) == "up" {
				up++
			} else {
				down++
			}
		}
		b.WriteString(fmt.Sprintf("*%s:*\n", escape(category.Name)))
		b.WriteString(fmt.Sprintf("`     ┌─ Up : %d`\n", up))
		b.WriteString(fmt.Sprintf("`     └─ Down : %d`\n", down))
	}

	if !hasContent {
		b.WriteString(escape("     - Tidak ada perangkat IPCN yang terdeteksi untuk gateway ini.\n"))
	}

	return b.String()
}

func formatModDemod(data GatewayData) string {
	var b strings.Builder
	b.WriteString("\n📶 *Modulator*\n")
	if data.DeviceProps != nil && len(data.DeviceProps.Data) > 0 {
		props := data.DeviceProps.Data[0]
		for i, mod := range props.Modulator {
			connector := "├"
			if i == len(props.Modulator)-1 {
				connector = "└"
			}
			b.WriteString(fmt.Sprintf("`     %s─ nIF%d : %d 🟩 || %d 🟥`\n", connector, mod.NifType, mod.Online, mod.Offline))
		}
		b.WriteString("\n📡 *Demodulator*\n")
		for i, demod := range props.Demodulator {
			connector := "├"
			if i == len(props.Demodulator)-1 {
				connector = "└"
			}
			b.WriteString(fmt.Sprintf("`     %s─ nIF%d : %d 🟩 || %d 🟥`\n", connector, demod.NifType, demod.Online, demod.Offline))
		}
	} else {
		b.WriteString(escape("\n📶 Modulator\n- Gagal mengambil data\n"))
		b.WriteString(escape("\n📡 Demodulator\n- Gagal mengambil data\n"))
	}
	return b.String()
}

func formatSatBeamInfo(data GatewayData) string {
	var b strings.Builder
	b.WriteString("\n🛰️ *Satellite & Beam Info*\n")
	if data.CnBeacon != nil {
		beaconStr := fmt.Sprintf("%.2f", data.CnBeacon.Data.Value)
		b.WriteString(fmt.Sprintf("`     ┌─ CN Beacon         : %s`\n", escape(beaconStr)))
	} else {
		b.WriteString(escape("`     ┌─ CN Beacon         : Gagal mengambil data`") + "\n")
	}

	if data.BeamStatus != nil {
		beam := data.BeamStatus.Data.StatusCounts
		b.WriteString(fmt.Sprintf("`     ├─ Beam Status       : %d 🟩 || %d 🟥`\n", beam.Online, beam.Offline))
		b.WriteString(fmt.Sprintf("`     └─ Satnet Status     : %d 🟩 || %d 🟥`\n", beam.Online, beam.Offline))
	} else {
		b.WriteString(escape("`     ├─ Beam Status       : Gagal mengambil data`") + "\n")
		b.WriteString(escape("`     └─ Satnet Status     : Gagal mengambil data`") + "\n")
	}
	return b.String()
}

func formatRtgsAiStatus(data GatewayData) string {
	var b strings.Builder
	b.WriteString("\n🤖 *RTGS AI Status*\n")
	if data.IntegratedStatus != nil {
		integrated := data.IntegratedStatus.Data
		b.WriteString(fmt.Sprintf("`     ┌─ Total Devices : %d`\n", integrated.Total))
		b.WriteString(fmt.Sprintf("`     ├─ 🟩            : %d`\n", integrated.Online))
		b.WriteString(fmt.Sprintf("`     └─ 🟥          : %d`\n", integrated.Offline))
	} else {
		b.WriteString(escape("- Gagal mengambil data\n"))
	}
	return b.String()
}

func FormatGatewaySummary(gatewayName string, data GatewayData) string {
	var b strings.Builder
	b.WriteString(FormatGatewayHeader(gatewayName))
	b.WriteString(formatSystemStatus(data))
	b.WriteString(formatTrafficInfo(data))
	b.WriteString(formatIpcnDeviceDetails(data, gatewayName))
	b.WriteString(formatModDemod(data))
	b.WriteString(formatSatBeamInfo(data))
	b.WriteString(formatRtgsAiStatus(data))
	return b.String()
}

func FormatIpTransitInfo(gatewayName string, status *api.IpcnStatusResponse, traffic *api.LnmIptxTrafficResponse, onlineUT *api.ToaRangeIntervalResponse) string {
	var b strings.Builder
	now := time.Now().Format("02 Jan 2006 15:04:05 WIB")
	b.WriteString(fmt.Sprintf("📡 *IP Transit Gateway %s*\n", escape(gatewayName)))
	b.WriteString(fmt.Sprintf("`      (%s)`\n\n", escape(now)))

	if status != nil {
		overall := "Down"
		if status.IpTransitMain.StatusText == "Up" || status.IpTransitBackupStatus.StatusText == "Up" {
			overall = "Up"
		}
		b.WriteString(fmt.Sprintf("`   ┌─ Overall Status     : %s`\n", escape(statusToEmoji(overall))))
		b.WriteString(fmt.Sprintf("`   ├─ IP Transit Main    : %s`\n", escape(statusToEmoji(status.IpTransitMain.StatusText))))
		b.WriteString(fmt.Sprintf("`   ├─ IP Transit Backup  : %s`\n", escape(statusToEmoji(status.IpTransitBackupStatus.StatusText))))
	} else {
		b.WriteString(escape("   - Gagal mengambil data status\n"))
	}

	if traffic != nil && len(traffic.HisData) > 0 {
		trafficStr := fmt.Sprintf("%.2f", traffic.HisData[0].TrafficTotalSpeed)
		b.WriteString(fmt.Sprintf("`   ├─ Current IPTX Traffic: %s Mbps`\n", escape(trafficStr)))
	} else {
		b.WriteString(escape("`   ├─ Current IPTX Traffic: Gagal mengambil data`") + "\n")
	}

	if onlineUT != nil && len(onlineUT.Data) > 0 {
		latestUT := onlineUT.Data[len(onlineUT.Data)-1]
		b.WriteString(fmt.Sprintf("`   └─ Current Online UT   : %d`\n", latestUT.UtOnlineToa))
	} else {
		b.WriteString(escape("`   └─ Current Online UT   : Gagal mengambil data`") + "\n")
	}
	return b.String()
}
