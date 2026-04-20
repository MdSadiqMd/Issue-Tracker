package pkg

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	internal "github.com/MdSadiqMd/issue-tracker/internal"
)

type WhatsAppMessage struct {
	ChatID  string `json:"chatId"`
	Message string `json:"message"`
}

func SendWhatsAppMessage(apiURL, chatID, message string) error {
	fmt.Printf("Sending WhatsApp message to %s\n", chatID)
	msgBody := WhatsAppMessage{
		ChatID:  chatID,
		Message: message,
	}

	bodyJSON, err := json.Marshal(msgBody)
	if err != nil {
		return fmt.Errorf("error marshaling message: %v", err)
	}

	headers := js.Global().Get("Object").New()
	headers.Set("Content-Type", "application/json")

	options := js.Global().Get("Object").New()
	options.Set("method", "POST")
	options.Set("headers", headers)
	options.Set("body", string(bodyJSON))

	fetchFunc := js.Global().Get("fetch")
	promise := fetchFunc.Invoke(apiURL, options)

	resultChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)

	thenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		status := response.Get("status").Int()

		if status != 200 {
			textPromise := response.Call("text")
			textPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				errorText := args[0].String()
				errorChan <- fmt.Errorf("Green API returned status %d: %s", status, errorText)
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
		fmt.Printf("WhatsApp message sent successfully: %s\n", string(data))
		return nil

	case err := <-errorChan:
		return fmt.Errorf("error sending WhatsApp message: %v", err)

	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout sending WhatsApp message")
	}
}

func FormatIssuesMessage(results []internal.RepoIssues) string {
	if len(results) == 0 {
		return "📊 *GitHub Issues Report*\n\nNo repositories tracked or no recent issues found."
	}

	message := "📊 *GitHub Issues Report*\n"
	message += fmt.Sprintf("_Generated at %s_\n\n", time.Now().Format("2006-01-02 15:04:05"))

	totalIssues := 0
	for _, result := range results {
		totalIssues += len(result.Issues)
	}

	message += fmt.Sprintf("*Total Issues: %d*\n", totalIssues)
	message += fmt.Sprintf("*Repositories: %d*\n\n", len(results))
	for _, result := range results {
		if len(result.Issues) == 0 {
			continue
		}

		message += fmt.Sprintf("🔹 *%s* (%d issues)\n", result.Repo, len(result.Issues))
		maxIssues := 5
		if len(result.Issues) < maxIssues {
			maxIssues = len(result.Issues)
		}

		for i := 0; i < maxIssues; i++ {
			issue := result.Issues[i]
			message += fmt.Sprintf("  • %s\n", issue.Title)
			message += fmt.Sprintf("    _%s_\n", issue.CreatedAt.Format("Jan 02, 15:04"))
		}
		if len(result.Issues) > maxIssues {
			message += fmt.Sprintf("  ... and %d more\n", len(result.Issues)-maxIssues)
		}

		message += "\n"
	}

	message += "---\n"
	return message
}
