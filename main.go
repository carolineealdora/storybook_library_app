package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"codebase/handler"
	"codebase/internal/services/subscription"
	"codebase/models"
)

func main() {
	shared := models.SharedHolder{}
	shared.Init()

	args := os.Args

	if len(args) > 1 {
		switch args[1] {
		case "migrate": // ponytail: schema is owned by storybook_management_app (its ADR-010)
			fmt.Println("migrations live in ../storybook_management_app: run `make migrate` there")
			return
		case "run":
		default:
			fmt.Println("unknown command")
			return
		}
	}
	run(&shared)
}

func run(shared *models.SharedHolder) {
	handler.InitRoute(shared)

	// Auto-renew job (PRD §12.1 Scheduler). Hourly so a renewal lands within the hour of expiry.
	go subscription.NewService(shared).RunRenewals(context.Background(), time.Hour)

	shared.Log.Info("Server starting on :" + shared.Config.Port)
	if err := shared.Echo.Start("0.0.0.0:" + shared.Config.Port); err != nil {
		shared.Log.Fatal("failed to start echo: ", err)
	}
}
