package types

type GatewayReport struct {
	FriendlyName string         `json:"friendly_name"`
	Satnets      []SatnetDetail `json:"satnets"`
}

type SatnetDetail struct {
	Name         string  `json:"name"`
	FwdTp        float64 `json:"fwd_tp"`
	RtnTp        float64 `json:"rtn_tp"`
	Time         string  `json:"time"`
	OnlineCount  *int64  `json:"online_count"`
	OfflineCount *int64  `json:"offline_count"`
}
