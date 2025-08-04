package api

import (
	"fmt"
	"net/url"
	"time"
)

func GetIpcnStatus(client *APIClient, baseURL string) (*IpcnStatusResponse, error) {
	var target IpcnStatusResponse
	err := client.GetWithAuth(fmt.Sprintf("%s/api/v1/ipcn/status", baseURL), &target)
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func GetIptxTraffic(client *APIClient, baseURL, sdate, edate, avg, gateway string) (*LnmIptxTrafficResponse, error) {
	var target LnmIptxTrafficResponse
	params := url.Values{}
	params.Add("sdate", sdate)
	params.Add("edate", edate)
	params.Add("avg", avg)
	params.Add("gateway", gateway)

	fullURL := fmt.Sprintf("%s/api/v1/lnm/prtg-data/iptx-traffic?%s", baseURL, params.Encode())
	err := client.GetWithAuth(fullURL, &target)
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func GetOnlineUT(client *APIClient, baseURL string) (*ToaRangeIntervalResponse, error) {
	var target ToaRangeIntervalResponse
	endDate := time.Now().UTC()
	startDate := endDate.Add(-1 * time.Hour)

	const layout = "2006-01-02T15:04:05"

	params := url.Values{}
	params.Add("start_date", startDate.Format(layout))
	params.Add("end_date", endDate.Format(layout))
	params.Add("interval", "60")

	fullURL := fmt.Sprintf("%s/api/v1/toa/range-interval?%s", baseURL, params.Encode())
	err := client.GetWithAuth(fullURL, &target)
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func GetIpcnSensorStatus(client *APIClient, baseURL string, deviceName string) (*IpcnSensorStatusResponse, error) {
	var target IpcnSensorStatusResponse
	fullURL := fmt.Sprintf("%s/api/v1/ipcn/sensor-status", baseURL)
	if deviceName != "" {
		fullURL += "?device_name=" + url.QueryEscape(deviceName)
	}
	err := client.GetWithAuth(fullURL, &target)
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func GetDevicePropertiesStatus(client *APIClient, baseURL string) (*DevicePropertiesStatusResponse, error) {
	var target DevicePropertiesStatusResponse
	err := client.GetWithAuth(fmt.Sprintf("%s/api/v1/device_properties/status", baseURL), &target)
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func GetCnBeacon(client *APIClient, baseURL string) (*CnBeaconResponse, error) {
	var target CnBeaconResponse
	err := client.GetWithAuth(fmt.Sprintf("%s/api/v1/lnm/cn_beacon", baseURL), &target)
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func GetBeamTerminalStatus(client *APIClient, baseURL string) (*TerminalBeamStatusResponse, error) {
	var target TerminalBeamStatusResponse
	err := client.GetWithAuth(fmt.Sprintf("%s/api/v1/terminal/beam-terminal-status", baseURL), &target)
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func GetTerminalStatusTotalIntegrated(client *APIClient, baseURL string) (*TerminalStatusTotalIntegratedResponse, error) {
	var target TerminalStatusTotalIntegratedResponse
	err := client.GetWithAuth(fmt.Sprintf("%s/api/v1/terminal/status/total/integrated", baseURL), &target)
	if err != nil {
		return nil, err
	}
	return &target, nil
}
