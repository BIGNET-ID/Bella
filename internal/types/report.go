package types

import "time"

type GatewayReport struct {
	FriendlyName string         `json:"friendly_name"`
	Satnets      []SatnetDetail `json:"satnets"`
}

type SatnetDetail struct {
	Name         string     `json:"name"`
	FwdTp        float64    `json:"fwd_tp"`
	RtnTp        float64    `json:"rtn_tp"`
	Time         string     `json:"time"`
	OnlineCount  *int64     `json:"online_count"`
	OfflineCount *int64     `json:"offline_count"`
	StartIssue   *time.Time `json:"start_issue"`
}

type SatnetUpAlert struct {
	GatewayName  string
	SatnetName   string
	RecoveryTime time.Time
	TimeDown     time.Time
}

type ModemDownAlert struct {
	GatewayName string
	DeviceName  string
	AlarmState  string
	StartTime   time.Time
}

type ModemUpAlert struct {
	GatewayName  string
	DeviceName   string
	RecoveryTime time.Time
	TimeDown     time.Time
}

type PRTGDownAlert struct {
	Location       string `json:"location"`
	SensorFullName string `json:"sensor_full_name"`
	DeviceName     string `json:"device_name"`
	SensorType     string `json:"sensor_type"`
	Value          string `json:"value"`
	Status         string `json:"status"`
	LastMessage    string `json:"last_message"`
	LastCheck      string `json:"last_check"`
	LastDown       string `json:"last_down,omitempty"`
	LastUp         string `json:"last_up,omitempty"`
}

type PRTGUpAlert struct {
	Location       string    `json:"location"`
	SensorFullName string    `json:"sensor_full_name"`
	DeviceName     string    `json:"device_name"`
	SensorType     string    `json:"sensor_type"`
	RecoveryTime   time.Time `json:"recovery_time"`
	LastDown       time.Time `json:"last_down"`
}
