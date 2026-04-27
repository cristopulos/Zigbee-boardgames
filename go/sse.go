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
// It implements a 30-second idle timeout on SSE reads to detect dead connections,
// and uses a goroutine-based read architecture to prevent blocking on network failures.
// Use [NewClient] to create an instance.
type Client struct {
	apiURL string
	client *http.Client
}

// NewClient creates a client for the given button-hub base URL.
func NewClient(apiURL string) *Client {
	return &Client{
		apiURL: strings.TrimSuffix(apiURL, "/"),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
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

	// Channels for async read results. Using a goroutine prevents blocking on
	// dead connections — the read call returns immediately, and we detect timeouts
	// via the select below.
	lineChan := make(chan []byte, 1)
	readErrChan := make(chan error, 1)

	// 30-second idle timeout: if no data arrives within this window, the SSE
	// connection is considered dead (e.g., network drop, server restart).
	// The timer is reset on every successful read to only measure idle time.
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	// resetTimer safely resets the timer, handling the edge case where Stop()
	// returns false (timer already fired) by draining the channel first.
	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(30 * time.Second)
	}

	// Read loop: spawns a goroutine for each read operation. This allows the
	// select to wake on either new data OR timeout, rather than blocking on
	// a hung TCP connection. Comment/ping lines (": ..." prefix) are skipped.
	for {
		go func() {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				readErrChan <- err
			} else {
				lineChan <- line
			}
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("sse: read timeout (no data received)")
		case line := <-lineChan:
			resetTimer()
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

			if bytes.HasPrefix(line, []byte(":")) {
				// Ignore comment/ping lines (keep-alive events)
				continue
			}

			if bytes.HasPrefix(line, []byte("data:")) {
				payload := bytes.TrimPrefix(line, []byte("data:"))
				payload = bytes.TrimLeft(payload, " ")
				data.Write(payload)
				data.WriteByte('\n')
			}
		case err := <-readErrChan:
			if data.Len() > 0 {
				var evt Event
				if err := json.Unmarshal(data.Bytes(), &evt); err == nil && evt.ButtonID == buttonID {
					handler(evt)
				}
			}
			return err
		}
	}
}
