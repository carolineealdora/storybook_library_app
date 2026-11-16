package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"codebase/handler"
	"codebase/internal/services/subscription"
	"codebase/migrations"
	"codebase/models"
)

func main() {
	shared := models.SharedHolder{}
	shared.Init()

	args := os.Args

	if len(args) > 1 {
		switch args[1] {
		case "migrate":
			migrate(shared, args)
			return
		case "run":
		default:
			fmt.Println("unknown command")
			return
		}
	}
	run(&shared)
}

func migrate(shared models.SharedHolder, args []string) {
	if len(args) < 3 {
		shared.Log.Error("usage: migrate up | down [name] | create <name>")
		return
	}
	migrator := migrations.NewMigration(shared)
	switch args[2] {
	case "up":
		err := migrator.Up()
		if err != nil {
			shared.Log.Error(err)
			return
		}
	case "down":
		var err error
		if len(args) <= 3 || len(args[3]) == 0 {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("[WARNING] Rollback All Migrations? (Y/N) ")
			char, _, _ := reader.ReadRune()
			if char == 'y' || char == 'Y' {
				err = migrator.Down("")
			} else {
				return
			}
		} else {
			err = migrator.Down(args[3])
		}
		if err != nil {
			shared.Log.Error(err)
			return
		}
	case "create":
		if len(args) <= 3 || len(args[3]) == 0 {
			shared.Log.Error("please input new migration name")
			return
		}

		err := migrator.Create(args[3])
		if err != nil {
			shared.Log.Error(err)
			return
		}
	default:
	}
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
