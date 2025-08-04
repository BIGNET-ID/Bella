package api

import "time"

type IpcnSensorStatus struct {
	DeviceName     string `json:"device_name"`
	StatustextPing string `json:"statustext_ping"`
}

type IpcnSensorStatusResponse []IpcnSensorStatus

type IpcnStatusResponse struct {
	IpTransitBackupStatus struct {
		StatusText string `json:"statustext"`
	} `json:"ip_transit_backup_status"`
	IpTransitMain struct {
		StatusText string `json:"statustext"`
	} `json:"ip_transit_main"`
	NifStatus struct {
		StatusText string `json:"statustext"`
	} `json:"nif_status"`
	NmsStatus struct {
		StatusText string `json:"statustext"`
	} `json:"nms_status"`
}

type LnmIptxTrafficResponse struct {
	HisData []struct {
		TrafficTotalSpeed float64 `json:"traffic_total_speed"`
	} `json:"hisdata"`
}

type ToaRangeIntervalResponse struct {
	Data []struct {
		UtOnlineToa int       `json:"ut_online_toa"`
		CreatedAt   time.Time `json:"created_at"`
	} `json:"data"`
}

type DevicePropertiesStatusResponse struct {
	Data []struct {
		Modulator []struct {
			NifType int `json:"nif_type"`
			Online  int `json:"online"`
			Offline int `json:"offline"`
		} `json:"modulator"`
		Demodulator []struct {
			NifType int `json:"nif_type"`
			Online  int `json:"online"`
			Offline int `json:"offline"`
		} `json:"demodulator"`
	} `json:"data"`
}

type CnBeaconResponse struct {
	Data struct {
		Value float64 `json:"value"`
	} `json:"data"`
}

type TerminalBeamStatusResponse struct {
	Data struct {
		StatusCounts struct {
			Offline int `json:"offline"`
			Online  int `json:"online"`
		} `json:"status_counts"`
	} `json:"data"`
}

type TerminalStatusTotalIntegratedResponse struct {
	Data struct {
		Total   int `json:"total"`
		Online  int `json:"online"`
		Offline int `json:"offline"`
	} `json:"data"`
}
