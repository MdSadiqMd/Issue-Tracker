package pkg

import (
	"context"
	"fmt"
	"syscall/js"
	"time"

	"github.com/syumai/workers/cloudflare"
	"github.com/syumai/workers/cloudflare/cron"
)

func CronTask(ctx context.Context) error {
	e, err := cron.NewEvent(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Cron task started at %v\n", e.ScheduledTime)

	cloudflare.WaitUntil(func() {
		url := "http://localhost:8787/send-report"

		headers := js.Global().Get("Object").New()
		headers.Set("Content-Type", "application/json")

		options := js.Global().Get("Object").New()
		options.Set("method", "POST")
		options.Set("headers", headers)

		fetchFunc := js.Global().Get("fetch")
		promise := fetchFunc.Invoke(url, options)

		resultChan := make(chan []byte, 1)
		errorChan := make(chan error, 1)

		thenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			response := args[0]
			status := response.Get("status").Int()

			if status != 200 {
				textPromise := response.Call("text")
				textPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					errorText := args[0].String()
					errorChan <- fmt.Errorf("endpoint returned status %d: %s", status, errorText)
					return nil
				}))
				return nil
			}

			jsonPromise := response.Call("json")
			jsonPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				jsonStr := js.Global().Get("JSON").Call("stringify", args[0]).String()
				resultChan <- []byte(jsonStr)
				return nil
			}))

			return nil
		})

		catchFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errorChan <- fmt.Errorf("fetch failed: %v", args[0].String())
			return nil
		})
		promise.Call("then", thenFunc)
		promise.Call("catch", catchFunc)

		select {
		case data := <-resultChan:
			fmt.Printf("Cron task completed successfully: %s\n", string(data))

		case err := <-errorChan:
			fmt.Printf("Cron task error: %v\n", err)

		case <-time.After(30 * time.Second):
			fmt.Printf("Cron task timeout\n")
		}
	})
	return nil
}
