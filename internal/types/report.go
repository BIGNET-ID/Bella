package types

// GatewayReport berisi detail untuk satu gateway yang memiliki masalah.
// Ini adalah struktur data JSON yang akan dihasilkan oleh Agent untuk SATU gateway.
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
	OnlineCount  *int64   `json:"online_count"`
	OfflineCount *int64   `json:"offline_count"`
}
