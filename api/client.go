package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type APIClient struct {
	HTTPClient *http.Client
	BaseURL    string
	Email      string
	Password   string
	Token      string
	mu         sync.Mutex
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

func NewAPIClient(baseURL, email, password string) *APIClient {
	return &APIClient{
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		BaseURL:    baseURL,
		Email:      email,
		Password:   password,
	}
}

func (c *APIClient) Login() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	slog.Info("Mencoba login untuk mendapatkan token baru...", "url", c.BaseURL)
	loginURL := fmt.Sprintf("%s/api/v1/auth/login", c.BaseURL)

	payload := LoginRequest{Email: c.Email, Password: c.Password}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal marshal payload login: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("gagal membuat request login: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal melakukan request login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login gagal dengan status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("gagal decode response login: %w", err)
	}

	if !loginResp.Status || loginResp.Data.Token == "" {
		return fmt.Errorf("login ke API tidak berhasil: %s", loginResp.Message)
	}

	c.Token = loginResp.Data.Token
	slog.Info("Berhasil login dan memperbarui token.")
	return nil
}

func (c *APIClient) GetWithAuth(url string, target interface{}) error {
	err := c.doGetRequest(url, target)

	if err != nil && strings.Contains(err.Error(), "status code tidak OK: 401") {
		slog.Warn("Menerima status 401 (Unauthorized), mencoba login ulang untuk refresh token...")

		if loginErr := c.Login(); loginErr != nil {
			return fmt.Errorf("gagal refresh token setelah error 401: %w", loginErr)
		}

		slog.Info("Mencoba ulang permintaan dengan token baru...", "url", url)
		err = c.doGetRequest(url, target)
	}

	return err
}

func (c *APIClient) doGetRequest(url string, target interface{}) error {
	if c.Token == "" {
		return fmt.Errorf("token otentikasi kosong, silakan login terlebih dahulu")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("gagal membuat request GET: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal melakukan request GET ke %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("menerima status code tidak OK: %d dari %s (dan gagal membaca body response: %v)", resp.StatusCode, url, readErr)
		}
		return fmt.Errorf("menerima status code tidak OK: %d dari %s. Response: %s", resp.StatusCode, url, string(bodyBytes))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
