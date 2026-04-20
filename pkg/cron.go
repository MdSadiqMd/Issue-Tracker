package pkg

import (
	"context"
	"fmt"

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

		headers := map[string]string{
			"Content-Type": "application/json",
		}

		data, err := FetchJS(url, "POST", headers, "")
		if err != nil {
			fmt.Printf("Cron task error: %v\n", err)
		} else {
			fmt.Printf("Cron task completed successfully: %s\n", string(data))
		}
	})
	return nil
}
