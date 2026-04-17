package gobutton

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client connects to the button-hub SSE stream.
// Use [NewClient] to create an instance.
type Client struct {
	apiURL string
	client *http.Client
}

// NewClient creates a client for the given button-hub base URL.
func NewClient(apiURL string) *Client {
	return &Client{
		apiURL: strings.TrimSuffix(apiURL, "/"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Listen opens the SSE stream and calls handler for every event whose
// button_id matches the supplied filter. It blocks until the context is
// cancelled or a non-recoverable error occurs.
// The caller is responsible for retry logic; see [Listen] for automatic reconnection.
func (c *Client) Listen(ctx context.Context, buttonID string, handler func(Event)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.apiURL+"/api/events/stream", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sse: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var data bytes.Buffer

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if data.Len() > 0 {
				var evt Event
				if err := json.Unmarshal(data.Bytes(), &evt); err == nil && evt.ButtonID == buttonID {
					handler(evt)
				}
			}
			return err
		}
		line = bytes.TrimRight(line, "\r\n")

		if len(line) == 0 {
			if data.Len() > 0 {
				var evt Event
				if err := json.Unmarshal(data.Bytes(), &evt); err == nil {
					if evt.ButtonID == buttonID {
						handler(evt)
					}
				}
				data.Reset()
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("data:")) {
			payload := bytes.TrimPrefix(line, []byte("data:"))
			payload = bytes.TrimLeft(payload, " ")
			data.Write(payload)
			data.WriteByte('\n')
		}
	}
}
