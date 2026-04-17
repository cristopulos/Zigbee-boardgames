package gobutton

// Package gobutton provides functions for registering and unregistering
// buttons with the button-hub API.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Register tells button-hub to recognise the given button_id.
// It is best-effort: failure is logged but not fatal.
func Register(ctx context.Context, apiURL, buttonID string) error {
	url := strings.TrimSuffix(apiURL, "/") + "/buttons"
	body, _ := json.Marshal(map[string]string{"button_id": buttonID})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register failed: %d %s", resp.StatusCode, string(rb))
	}
	return nil
}

// Unregister removes the button_id from button-hub.
func Unregister(ctx context.Context, apiURL, buttonID string) error {
	url := fmt.Sprintf("%s/buttons/%s", strings.TrimSuffix(apiURL, "/"), buttonID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unregister failed: %d %s", resp.StatusCode, string(rb))
	}
	return nil
}
