//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"syscall/js"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/web"
	"github.com/megascan/gpui-go/scheduler"
)

const (
	readerCount = 8
	bodyLength  = 32768 * 3
)

func verifyBody(body []byte) error {
	if len(body) != bodyLength {
		return fmt.Errorf("response did not reach EOF")
	}
	for index := 0; index < len(body)/3; index++ {
		expected := []byte{byte(index % 251), byte(index % 239), byte(index % 227)}
		if !bytes.Equal(body[index*3:index*3+3], expected) {
			return fmt.Errorf("response chunks must remain ordered")
		}
	}
	return nil
}

func readOne(s *scheduler.Scope, client web.FetchHTTPClient, url string) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Send(request).Await(s)
	if err != nil {
		return err
	}
	body, err := readBodyOffTask(s, response.Body)
	if err != nil {
		return err
	}
	return verifyBody(body)
}

func fetchAll(s *scheduler.Scope, platform *web.WebPlatform, client web.FetchHTTPClient, url string) error {
	tasks := make([]*gpui.Task[struct{}], 0, readerCount)
	for range readerCount {
		tasks = append(tasks, platform.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
			return struct{}{}, readOne(s, client, url)
		}))
	}
	for _, task := range tasks {
		if _, err := task.Await(s); err != nil {
			return err
		}
	}
	return nil
}

func registerHTTPClientTest() {
	exports := js.Global().Get("Object").New()
	exports.Set("test_worker_fetch", js.FuncOf(func(_ js.Value, args []js.Value) any {
		url := args[0].String()
		return js.Global().Get("Promise").New(js.FuncOf(func(_ js.Value, settle []js.Value) any {
			resolve, reject := settle[0], settle[1]
			platform := web.NewWebPlatform(true)
			client := web.NewFetchHTTPClient(platform.ForegroundExecutor().ToSend())
			platform.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
				if err := fetchAll(s, platform, client, url); err != nil {
					reject.Invoke(err.Error())
				} else {
					resolve.Invoke()
				}
				return struct{}{}, nil
			}).Detach()
			return nil
		}))
	}))
	js.Global().Set("gpuiWebTests", exports)
}
