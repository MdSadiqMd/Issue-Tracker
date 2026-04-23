package tasks

import (
	"context"
	"fmt"

	internal "github.com/MdSadiqMd/issue-tracker/internal"
	"github.com/MdSadiqMd/issue-tracker/pkg"
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
		cfg, err := internal.GetGreenAPIConfig()
		if err != nil {
			fmt.Printf("Cron task error: %v\n", err)
			return
		}

		results, err := internal.FetchIssuesLogic()
		if err != nil {
			fmt.Printf("Error fetching issues: %v\n", err)
			errorMsg := fmt.Sprintf("Error fetching GitHub issues\n\n%v", err)
			if sendErr := pkg.SendWhatsAppMessage(cfg.APIURL, cfg.IdInstance, cfg.ApiTokenInstance, cfg.ChatID, errorMsg); sendErr != nil {
				fmt.Printf("Failed to send cron error notification: %v\n", sendErr)
			}
			return
		}

		totalIssues := 0
		for _, r := range results {
			totalIssues += len(r.Issues)
		}

		if totalIssues == 0 {
			fmt.Println("No issues found, skipping WhatsApp notification")
			return
		}

		message := internal.FormatIssuesMessage(results)
		if err := pkg.SendWhatsAppMessage(cfg.APIURL, cfg.IdInstance, cfg.ApiTokenInstance, cfg.ChatID, message); err != nil {
			fmt.Printf("Error sending WhatsApp message: %v\n", err)
			return
		}

		fmt.Printf("Cron task completed successfully: sent %d issue(s)\n", totalIssues)
	})
	return nil
}
