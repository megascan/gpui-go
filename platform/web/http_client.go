//go:build js && wasm

package web

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

// FetchHTTPClient is a gpui.HTTPClient over the browser Fetch API. Its response bodies are
// read from goroutines: in Go wasm a Read that waits for the network inside a task or a
// browser callback deadlocks the page.
type FetchHTTPClient struct {
	foregroundExecutor gpui.SendForegroundExecutor
	userAgent          gpui.Option[string]
	credentials        FetchCredentials
}

var _ gpui.HTTPClient = FetchHTTPClient{}

// FetchCredentials controls whether browser Fetch requests include credentials.
type FetchCredentials uint8

const (
	// FetchCredentialsOmit never sends credentials in the request or includes them in the response.
	FetchCredentialsOmit FetchCredentials = iota
	// FetchCredentialsSameOrigin sends and includes credentials for same-origin requests only;
	// it is the default.
	FetchCredentialsSameOrigin
	// FetchCredentialsInclude always includes credentials, even for cross-origin requests.
	FetchCredentialsInclude
)

// DefaultFetchCredentials returns FetchCredentialsSameOrigin.
func DefaultFetchCredentials() FetchCredentials {
	return FetchCredentialsSameOrigin
}

// String formats the variant as upstream's Debug does.
func (c FetchCredentials) String() string {
	switch c {
	case FetchCredentialsOmit:
		return "Omit"
	case FetchCredentialsInclude:
		return "Include"
	}
	return "SameOrigin"
}

// NewFetchHTTPClient creates a browser Fetch client using the web platform's foreground executor.
func NewFetchHTTPClient(foregroundExecutor gpui.SendForegroundExecutor) FetchHTTPClient {
	return FetchHTTPClient{
		foregroundExecutor: foregroundExecutor,
		userAgent:          gpui.None[string](),
		credentials:        DefaultFetchCredentials(),
	}
}

// FetchHTTPClientWithUserAgent creates a browser Fetch client with the given reported user agent.
func FetchHTTPClientWithUserAgent(foregroundExecutor gpui.SendForegroundExecutor, userAgent string) (FetchHTTPClient, error) {
	if err := validateHeaderValue(userAgent); err != nil {
		return FetchHTTPClient{}, err
	}
	return FetchHTTPClient{
		foregroundExecutor: foregroundExecutor,
		userAgent:          gpui.Some(userAgent),
		credentials:        DefaultFetchCredentials(),
	}, nil
}

// WithCredentials returns the client with credentials.
func (c FetchHTTPClient) WithCredentials(credentials FetchCredentials) FetchHTTPClient {
	c.credentials = credentials
	return c
}

// UserAgent returns the reported user agent, if any.
func (c FetchHTTPClient) UserAgent() (string, bool) {
	return c.userAgent.Get()
}

// Proxy returns false: the browser chooses proxies.
func (c FetchHTTPClient) Proxy() (*url.URL, bool) {
	return nil, false
}

// Send buffers req's body and fetches it on the main thread.
func (c FetchHTTPClient) Send(req *http.Request) *gpui.Task[*http.Response] {
	credentials := c.credentials
	return c.foregroundExecutor.Spawn(func(s *scheduler.Scope) (*http.Response, error) {
		bodyBytes, err := readBodyToBytes(req.Body)
		if err != nil {
			return nil, err
		}
		return fetch(s, req, bodyBytes, credentials)
	})
}

var errInvalidHeaderValue = errors.New("failed to parse header value")

func validateHeaderValue(value string) error {
	for i := 0; i < len(value); i++ {
		b := value[i]
		if (b < 32 && b != '\t') || b == 127 {
			return errInvalidHeaderValue
		}
	}
	return nil
}

func headerValueToStr(value string) bool {
	for i := 0; i < len(value); i++ {
		b := value[i]
		if !(b == '\t' || (b >= 32 && b < 127)) {
			return false
		}
	}
	return true
}

func fetch(s *scheduler.Scope, req *http.Request, bodyBytes []byte, credentials FetchCredentials) (*http.Response, error) {
	init := js.Global().Get("Object").New()
	init.Set("method", req.Method)
	switch credentials {
	case FetchCredentialsOmit:
		init.Set("credentials", "omit")
	case FetchCredentialsSameOrigin:
		init.Set("credentials", "same-origin")
	case FetchCredentialsInclude:
		init.Set("credentials", "include")
	}

	if bodyBytes != nil {
		uint8array := js.Global().Get("Uint8Array").New(len(bodyBytes))
		js.CopyBytesToJS(uint8array, bodyBytes)
		init.Set("body", uint8array)
	}

	requestURL := req.URL.String()
	request, err := jsNew(js.Global().Get("Request"), requestURL, init)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetch Request: %s", jsErrorDebug(err))
	}

	requestHeaders := request.Get("headers")
	names := make([]string, 0, len(req.Header))
	for name := range req.Header {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		lower := strings.ToLower(name)
		for _, value := range req.Header[name] {
			if !headerValueToStr(value) {
				return nil, fmt.Errorf("non-ASCII header value for %s", lower)
			}
			if _, err := jsCall(requestHeaders, "set", lower, value); err != nil {
				return nil, fmt.Errorf("failed to set header %s: %s", lower, jsErrorDebug(err))
			}
		}
	}

	promise, err := jsCall(js.Global(), "fetch", request)
	if err != nil {
		return nil, fmt.Errorf("fetch threw an error: %s", jsErrorDebug(err))
	}
	responseValue, err := awaitPromise(s, promise)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %s", jsErrorDebug(err))
	}

	if !jsInstanceOf(responseValue, "Response") {
		return nil, fmt.Errorf("fetch result is not a Response: %s", jsDebug(responseValue))
	}
	webResponse := responseValue

	status := webResponse.Get("status").Int()
	header := http.Header{}

	headerPairs := js.Global().Get("Array").Call("from", webResponse.Get("headers"))
	length := headerPairs.Length()
	for index := range length {
		entry := headerPairs.Index(index)
		if !js.Global().Get("Array").Call("isArray", entry).Bool() {
			slog.Warn(fmt.Sprintf("skipping non-array header entry at index %d: %s", index, jsDebug(entry)))
			continue
		}
		name, nameOK := jsString(entry.Index(0))
		value, valueOK := jsString(entry.Index(1))
		if nameOK && valueOK {
			header.Add(name, value)
			continue
		}
		slog.Warn(fmt.Sprintf("skipping response header at index %d: name=%s, value=%s",
			index, debugOptionString(name, nameOK), debugOptionString(value, valueOK)))
	}

	var body io.ReadCloser
	stream := webResponse.Get("body")
	if !jsIsNullish(stream) {
		reader, err := jsCall(stream, "getReader")
		if err != nil || !jsInstanceOf(reader, "ReadableStreamDefaultReader") {
			if err == nil {
				err = jsRejection{reader}
			}
			return nil, fmt.Errorf("response body reader has an unexpected type: %s", jsErrorDebug(err))
		}
		body = newReadableStreamBody(reader)
	} else {
		body = http.NoBody
	}

	if status < 100 || status > 999 {
		body.Close()
		return nil, errors.New("invalid status code")
	}

	return &http.Response{
		Status:        strconv.Itoa(status) + " " + http.StatusText(status),
		StatusCode:    status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          body,
		ContentLength: -1,
		Request:       req,
	}, nil
}

func debugOptionString(v string, ok bool) string {
	if !ok {
		return "None"
	}
	return "Some(" + strconv.Quote(v) + ")"
}

func readBodyToBytes(body io.ReadCloser) ([]byte, error) {
	if body == nil || body == http.NoBody {
		return nil, nil
	}
	defer body.Close()
	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, body); err != nil {
		return nil, err
	}
	if buffer.Len() == 0 {
		return nil, nil
	}
	return buffer.Bytes(), nil
}

const responseBodyChannelCapacity = 8

type responseChunk struct {
	data []byte
	err  error
}

type readableStreamBody struct {
	chunks       chan responseChunk
	cancellation chan struct{}
	cancelOnce   sync.Once
	pending      []byte
	err          error
}

func newReadableStreamBody(reader js.Value) *readableStreamBody {
	b := &readableStreamBody{
		chunks:       make(chan responseChunk, responseBodyChannelCapacity),
		cancellation: make(chan struct{}),
	}
	go pumpResponseBody(reader, b.chunks, b.cancellation)
	return b
}

// Read reads the next bytes of the response; it must not be called from a task or a browser
// callback.
func (b *readableStreamBody) Read(buffer []byte) (int, error) {
	for len(b.pending) == 0 {
		if b.err != nil {
			return 0, b.err
		}
		chunk, ok := <-b.chunks
		if !ok {
			b.err = io.EOF
			continue
		}
		if chunk.err != nil {
			b.err = chunk.err
			continue
		}
		b.pending = chunk.data
	}
	n := copy(buffer, b.pending)
	b.pending = b.pending[n:]
	return n, nil
}

// Close cancels the browser-side stream.
func (b *readableStreamBody) Close() error {
	b.cancelOnce.Do(func() { close(b.cancellation) })
	return nil
}

func pumpResponseBody(reader js.Value, chunks chan<- responseChunk, cancellation <-chan struct{}) {
	defer close(chunks)
	for {
		read := make(chan promiseOutcome, 1)
		promise, err := jsCall(reader, "read")
		if err == nil {
			err = promiseThen(promise, func(outcome promiseOutcome) { read <- outcome })
		}
		if err != nil {
			read <- promiseOutcome{value: js.ValueOf(err.Error()), rejected: true}
		}
		var outcome promiseOutcome
		select {
		case <-cancellation:
			cancelReader(reader)
			return
		case outcome = <-read:
		}

		var chunk []byte
		var chunkErr error
		if outcome.rejected {
			chunkErr = fmt.Errorf("response stream failed: %s", jsDebug(outcome.value))
		} else {
			chunk, chunkErr = responseChunkOf(outcome.value)
		}
		switch {
		case chunkErr == nil && chunk != nil:
			select {
			case chunks <- responseChunk{data: chunk}:
			case <-cancellation:
				cancelReader(reader)
				return
			}
		case chunkErr == nil:
			return
		default:
			select {
			case chunks <- responseChunk{err: chunkErr}:
			case <-cancellation:
				slog.Debug("response body receiver was dropped after a stream error")
			}
			return
		}
	}
}

func responseChunkOf(result js.Value) ([]byte, error) {
	if done, _ := jsBool(jsGet(result, "done")); done {
		return nil, nil
	}
	value := jsGet(result, "value")
	if !jsInstanceOf(value, "Uint8Array") {
		return nil, fmt.Errorf("response stream yielded a non-byte chunk: %s", jsDebug(value))
	}
	chunk := make([]byte, value.Length())
	js.CopyBytesToGo(chunk, value)
	return chunk, nil
}

func cancelReader(reader js.Value) {
	promise, err := jsCall(reader, "cancel")
	if err == nil {
		_, err = awaitPromiseBlocking(promise)
	}
	if err != nil {
		slog.Debug(fmt.Sprintf("failed to cancel response body reader: %s", jsErrorDebug(err)))
	}
}
