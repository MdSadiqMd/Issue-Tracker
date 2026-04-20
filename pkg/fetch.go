package pkg

import (
	"fmt"
	"syscall/js"
	"time"
)

func FetchJS(url, method string, headers map[string]string, body string) ([]byte, error) {
	jsHeaders := js.Global().Get("Object").New()
	for k, v := range headers {
		jsHeaders.Set(k, v)
	}

	options := js.Global().Get("Object").New()
	options.Set("method", method)
	options.Set("headers", jsHeaders)
	if body != "" {
		options.Set("body", body)
	}

	fetchFunc := js.Global().Get("fetch")
	promise := fetchFunc.Invoke(url, options)

	resultChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)
	doneChan := make(chan struct{})

	var thenFunc, catchFunc js.Func

	thenFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		status := response.Get("status").Int()

		if status < 200 || status >= 300 {
			var textPromiseThen js.Func
			textPromiseThen = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				errorChan <- fmt.Errorf("API returned status %d: %s", status, args[0].String())
				close(doneChan)
				textPromiseThen.Release()
				return nil
			})
			response.Call("text").Call("then", textPromiseThen)
			return nil
		}

		var textPromiseThen js.Func
		textPromiseThen = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resultChan <- []byte(args[0].String())
			close(doneChan)
			textPromiseThen.Release()
			return nil
		})
		response.Call("text").Call("then", textPromiseThen)

		return nil
	})

	catchFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errorChan <- fmt.Errorf("fetch failed: %v", args[0].String())
		close(doneChan)
		return nil
	})

	promise.Call("then", thenFunc)
	promise.Call("catch", catchFunc)

	go func() {
		<-doneChan
		thenFunc.Release()
		catchFunc.Release()
	}()

	select {
	case data := <-resultChan:
		return data, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("fetch timeout for %s", url)
	}
}
