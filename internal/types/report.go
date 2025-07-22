package types

// GatewayReport adalah struktur data utama untuk laporan per gateway.
type GatewayReport struct {
	FriendlyName string         `json:"friendly_name"`
	Satnets      []SatnetDetail `json:"satnets"`
}

// SatnetDetail berisi semua data yang dikumpulkan untuk satu satnet.
type SatnetDetail struct {
	Name         string  `json:"name"`
	FwdTp        float64 `json:"fwd_tp"`
	RtnTp        float64 `json:"rtn_tp"`
	Time         string  `json:"time"` // Waktu dari data satnet (sudah diformat)
	// Menggunakan pointer agar bisa membedakan antara nilai 0 dan null (tidak ada data).
	OnlineCount  *int64 `json:"online_count"`
	OfflineCount *int64 `json:"offline_count"`
}
