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
// No HTTP timeout is set — SSE streams are long-lived and timeouts would
// kill healthy connections. Use ctx for cancellation instead.
func NewClient(apiURL string) *Client {
	return &Client{
		apiURL: strings.TrimSuffix(apiURL, "/"),
		client: &http.Client{},
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

	// Channels for async read results. A single persistent goroutine (not
	// one per iteration) pumps lines into lineChan so the select can wake on
	// either new data OR the idle timer, without blocking on a dead connection.
	lineChan := make(chan []byte, 1)
	readErrChan := make(chan error, 1)

	// Persistent read goroutine — spawned once, lives until the connection
	// closes or the context is cancelled. Closing resp.Body (triggered by ctx
	// cancellation) unblocks the in-flight ReadBytes.
	// Non-blocking sends on lineChan and readErrChan prevent the goroutine
	// from leaking if the main loop returns while it's trying to send.
	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				select {
				case readErrChan <- err:
				default:
				}
				return
			}
			select {
			case lineChan <- line:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Body-close done channel: ensures the body-close goroutine exits when
	// Listen returns (not just on ctx cancellation), preventing double-close
	// with the defer and preventing goroutine accumulation across reconnects.
	bodyCloseDone := make(chan struct{})
	defer close(bodyCloseDone)

	// Separate goroutine: close the body when the context is cancelled so
	// the read goroutine unblocks promptly. Also exits when bodyCloseDone
	// is closed to prevent leak on normal return.
	go func() {
		select {
		case <-ctx.Done():
			resp.Body.Close()
		case <-bodyCloseDone:
		}
	}()

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

	// Main event loop: select waits for ctx cancellation, idle timeout,
	// new lines from the read goroutine, or read errors.
	var data bytes.Buffer
	for {
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
				// Ignore comment/ping lines (keep-alive events from server)
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
