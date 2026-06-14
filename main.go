package main

import (
	"bufio"
	"fmt"
	"os"

	"codebase/handler"
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
			run(shared)
			return
		default:
			fmt.Println("unknown command")
			return
		}
	}
	handler.InitRoute(&shared)
	run(shared)
}

func migrate(shared models.SharedHolder, args []string) {
	migrator := migrations.NewMigration(shared)
	switch args[2] {
	case "up":
		err := migrator.Up()
		if err != nil {
			shared.Log.Error(err)
			return
		}
		break
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
		break
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
		break
	default:
		break
	}
}

func run(shared models.SharedHolder) {
	shared.Log.Info("Server started")
}
