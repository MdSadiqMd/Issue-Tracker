package pkg

import (
	"context"
	"fmt"

	"github.com/MdSadiqMd/issue-tracker/internal"
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
		_, err := internal.FetchIssuesLogic()
		if err != nil {
			fmt.Printf("Cron task error: %v\n", err)
		} else {
			fmt.Println("Cron task completed successfully")
		}
	})
	return nil
}
