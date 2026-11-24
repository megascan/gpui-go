package support

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

const requestTimeout = 60 * time.Second

const maxResponseBodyBytes = 256 << 20

// HTTPClient is a gpui.HTTPClient over net/http, the examples' stand-in for reqwest_client.
type HTTPClient struct {
	client    *http.Client
	userAgent string
	executor  gpui.BackgroundExecutor
}

// NewHTTPClientWithUserAgent creates an HTTPClient that sends userAgent and spawns its
// requests on executor.
func NewHTTPClientWithUserAgent(userAgent string, executor gpui.BackgroundExecutor) (*HTTPClient, error) {
	return &HTTPClient{client: &http.Client{Timeout: requestTimeout}, userAgent: userAgent, executor: executor}, nil
}

// Send sends req off the scheduler and yields the response with its body read into memory.
func (c *HTTPClient) Send(req *http.Request) *gpui.Task[*http.Response] {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	type outcome struct {
		response *http.Response
		err      error
	}
	tx, rx := scheduler.Oneshot[outcome]()
	go func() {
		response, err := c.client.Do(req)
		if err == nil {
			var body []byte
			body, err = readBoundedBody(response.Body)
			_ = response.Body.Close()
			response.Body = io.NopCloser(bytes.NewReader(body))
		}
		_ = tx.Send(outcome{response: response, err: err})
	}()
	return c.executor.Spawn(func(s *scheduler.Scope) (*http.Response, error) {
		result, err := rx.Recv(s)
		if err != nil {
			return nil, err
		}
		if result.err != nil {
			return nil, fmt.Errorf("sending request: %w", result.err)
		}
		return result.response, nil
	})
}

func readBoundedBody(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxResponseBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBodyBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxResponseBodyBytes)
	}
	return body, nil
}

// UserAgent returns the user agent the client sends.
func (c *HTTPClient) UserAgent() (string, bool) {
	return c.userAgent, true
}

// Proxy returns no proxy.
func (c *HTTPClient) Proxy() (*url.URL, bool) {
	return nil, false
}
