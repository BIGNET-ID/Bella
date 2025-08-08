package state

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

type ActiveAlert struct {
	Type    string      `json:"type"`
	Gateway string      `json:"gateway"`
	Details interface{} `json:"details"`
}

type Manager struct {
	filePath     string
	mu           sync.Mutex
	activeAlerts map[string]ActiveAlert
}

func NewManager(filePath string) *Manager {
	m := &Manager{
		filePath:     filePath,
		activeAlerts: make(map[string]ActiveAlert),
	}
	if err := m.load(); err != nil {
		slog.Warn("Tidak dapat memuat file status, memulai dengan status kosong.", "file", filePath, "error", err)
	}
	return m
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &m.activeAlerts)
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.activeAlerts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0644)
}

func (m *Manager) GetActiveAlerts() map[string]ActiveAlert {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := make(map[string]ActiveAlert)
	for k, v := range m.activeAlerts {
		clone[k] = v
	}
	return clone
}

func (m *Manager) AddAlert(key string, alert ActiveAlert) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeAlerts[key] = alert
	if err := m.save(); err != nil {
		slog.Error("Gagal menyimpan file status setelah menambah alert", "file", m.filePath, "key", key, "error", err)
	}
}

func (m *Manager) RemoveAlertByKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.activeAlerts[key]; exists {
		delete(m.activeAlerts, key)
		if err := m.save(); err != nil {
			slog.Error("Gagal menyimpan file status setelah menghapus alert", "file", m.filePath, "key", key, "error", err)
		}
	}
}


func (m *Manager) GetAlertByKey(key string) (ActiveAlert, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert, exists := m.activeAlerts[key]
	return alert, exists
}