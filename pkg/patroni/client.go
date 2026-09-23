package patroni

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// --- Patroni API Data Structures ---

type Member struct {
	Name           string `json:"name"`
	Role           string `json:"role"` // leader, sync_standby, replica, demoted
	State          string `json:"state"` // running, streaming, stopped
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Timeline       int    `json:"timeline"`
	Lag            uint64 `json:"lag,omitempty"`
	PendingRestart bool   `json:"pending_restart,omitempty"`
}

type ClusterStatus struct {
	Scope   string   `json:"scope"`
	Members []Member `json:"members"`
	ScheduledSwitchover *struct {
		At    string `json:"at"`
		From  string `json:"from"`
		To    string `json:"to"`
	} `json:"scheduled_switchover,omitempty"`
	Pause bool `json:"pause"`
}

type SwitchoverPayload struct {
	Leader    string `json:"leader,omitempty"`
	Candidate string `json:"candidate,omitempty"`
	Scheduled string `json:"scheduled,omitempty"` // ISO 8601 string or empty for immediate
}

// --- Patroni Client ---

type Client struct {
	httpClient *http.Client
	endpoints  []string
	mu         sync.RWMutex
}

func NewClient(initialEndpoints []string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		endpoints: initialEndpoints,
	}
}

// GetClusterState запрашивает статус кластера. Если основной URL недоступен, фолбэчится на другие известные узлы.
func (c *Client) GetClusterState(ctx context.Context) (*ClusterStatus, error) {
	c.mu.RLock()
	endpoints := make([]string, len(c.endpoints))
	copy(endpoints, c.endpoints)
	c.mu.RUnlock()

	var lastErr error
	for _, ep := range endpoints {
		url := fmt.Sprintf("%s/cluster", ep)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("node %s returned status %d", ep, resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		var status ClusterStatus
		if err := json.Unmarshal(body, &status); err != nil {
			lastErr = err
			continue
		}

		// Auto-discovery: обновляем список эндпоинтов на основе ответа Patroni
		c.updateEndpointsFromStatus(&status)

		return &status, nil
	}

	return nil, fmt.Errorf("failed to fetch cluster status from all endpoints: %w", lastErr)
}

// updateEndpointsFromStatus автоматически добавляет найденные ноды в список опроса
func (c *Client) updateEndpointsFromStatus(status *ClusterStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()

	newEndpoints := make([]string, 0, len(status.Members))
	for _, m := range status.Members {
		// Обычно Patroni API висит на порту 8008, но если настроен иной порт, его можно переопределить
		ep := fmt.Sprintf("http://%s:8008", m.Host)
		newEndpoints = append(newEndpoints, ep)
	}

	if len(newEndpoints) > 0 {
		c.endpoints = newEndpoints
	}
}

// Switchover инициирует передачу роли Primary другому узлу
func (c *Client) Switchover(ctx context.Context, leaderNode, candidateNode string) error {
	payload := SwitchoverPayload{
		Leader:    leaderNode,
		Candidate: candidateNode,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	c.mu.RLock()
	targetEp := c.endpoints[0] // Отправляем на любую доступную ноду
	c.mu.RUnlock()

	url := fmt.Sprintf("%s/switchover", targetEp)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("switchover failed (code %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Reinitialize перезапускает синхронизацию реплики с лидером
func (c *Client) Reinitialize(ctx context.Context, targetNode string) error {
	c.mu.RLock()
	endpoints := c.endpoints
	c.mu.RUnlock()

	var targetURL string
	// Ищем эндпоинт целевой ноды
	for _, ep := range endpoints {
		if ep == fmt.Sprintf("http://%s:8008", targetNode) {
			targetURL = ep
			break
		}
	}
	if targetURL == "" && len(endpoints) > 0 {
		targetURL = endpoints[0]
	}

	url := fmt.Sprintf("%s/reinitialize", targetURL)
	payload := map[string]string{"subcluster": "", "force": "true"}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reinitialize failed (code %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
