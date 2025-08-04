package bot

import (
	"bella/api"
	"strings"
)

type DeviceCategory struct {
	Name    string
	Devices []api.IpcnSensorStatus
}

var gatewayDeviceMapping = map[string]map[string][]string{
	"Timika": {
		"Core Router":       {"IPCN_TMK_CR2-G1L", "IPCN_TMK_CR1-G1L"},
		"Core Switch":       {"IPCN_TMK_CSW-G1L"},
		"Management Router": {"IPCN_TMK_MR1-G1L", "IPCN_TMK_MR2-G1L"},
		"Management Switch": {"IPCN_TMK_MSW-G1L"},
		"Firewall":          {"IPCN_TMK_NGFW2-G1L", "IPCN_TMK_NGFW1-G1L"},
		"Sandvine":          {"IPCN_TMK_DPI-G1L"},
		"Server":            {"IPCN_TMK_SRV2-G1L"},
	},
	"Manokwari": {
		"Core Router":       {"IPCN_MNK_CR2-G1K", "IPCN_MNK_CR1-G1K"},
		"Core Switch":       {"IPCN_MNK_CSW-G1K"},
		"Management Router": {"IPCN_MNK_MR1-G1K", "IPCN_MNK_MR2-G1K"},
		"Management Switch": {"IPCN_MNK_MSW-G1K"},
		"Firewall":          {"IPCN_MNK_NGFW1-G1K", "IPCN_MNK_NGFW2-G1K"},
		"Sandvine":          {"IPCN_MNK_DPI-G1K"},
	},
	"Jayapura": {
		"Core Router":       {"IPCN_JYP_G1G-CR2", "IPCN_JYP_G1G-CR1"},
		"Core Switch":       {"IPCN_JYP_G1G-CSW2", "IPCN_JYP_G1G-CSW1"},
		"Management Router": {"IPCN_JYP_G1G-MR2", "IPCN_JYP_G1G-MR1", "IPCN_JYP_G1G-MR3", "IPCN_JYP_G1G-MR4"},
		"Management Switch": {"IPCN_JYP_G1G-MSW"},
		"Firewall":          {"IPCN_JYP_G1G-NGFW2", "IPCN_JYP_G1G-NGFW1", "IPCN_JYP_G1G-NGFW3"},
		"CHR Mikrotik":      {"IPCN_JYP_G1G-CICI2", "IPCN_JYP_G1G-CICI1"},
		"Server":            {"IPCN_JYP_G1G-SRV01", "IPCN_JYP_G1G-SRV02"},
	},
}

func getDeviceCategoriesForGateway(gatewayName string) map[string]*DeviceCategory {
	categories := make(map[string]*DeviceCategory)

	deviceMap, ok := gatewayDeviceMapping[gatewayName]
	if !ok {
		return categories
	}

	order := []string{"Core Router", "Core Switch", "Management Router", "Management Switch", "Firewall", "VPN Gateway", "CHR Mikrotik", "Sandvine", "Server"}
	for _, categoryName := range order {
		if _, exists := deviceMap[categoryName]; exists {
			categories[categoryName] = &DeviceCategory{Name: categoryName}
		}
	}

	return categories
}

func categorizeSensors(sensors *api.IpcnSensorStatusResponse, gatewayName string) map[string]*DeviceCategory {
	categories := getDeviceCategoriesForGateway(gatewayName)
	if sensors == nil || len(categories) == 0 {
		return categories
	}

	deviceToCategoryMap := make(map[string]string)
	deviceMapForGateway := gatewayDeviceMapping[gatewayName]

	for categoryName, deviceNames := range deviceMapForGateway {
		for _, deviceName := range deviceNames {
			deviceToCategoryMap[strings.TrimSpace(deviceName)] = categoryName
		}
	}

	for _, sensor := range *sensors {
		trimmedDeviceName := strings.TrimSpace(sensor.DeviceName)
		if trimmedDeviceName == "" {
			continue
		}

		if categoryName, found := deviceToCategoryMap[trimmedDeviceName]; found {
			if category, ok := categories[categoryName]; ok {
				category.Devices = append(category.Devices, sensor)
			}
		}
	}

	return categories
}