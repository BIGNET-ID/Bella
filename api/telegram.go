package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var client = &http.Client{Timeout: 60 * time.Second}

func GetJSON(url string, target interface{}) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("gagal melakukan permintaan ke %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("menerima status code tidak OK: %d dari %s", resp.StatusCode, url)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}