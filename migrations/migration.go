package migrations

import (
	"os"

	"codebase/models"
	"codebase/pkg/migrator/postgres"
	sql "codebase/pkg/orm"
)

var (
	scripts    = map[string]postgres.SQLMigration{}
	migrations = []string{
		"20261116000001_create_storybook_tables",
		"20261116000002_seed_mock_accounts_and_plan",
		"20261116000003_add_forgot_pin_attempt_limit",
	}
)

func NewMigration(holder models.SharedHolder) postgres.Migrator {
	for _, migration := range migrations {
		scripts[migration] = postgres.SQLMigration{
			Up: LoadSQLScript("./migrations/" + migration + ".sql"),
		}
	}

	migrator := postgres.NewMigrator(holder.Sql, "migrations", "./migrations/", scripts)

	if err := migrator.Init(); err != nil {
		holder.Log.Debug(err)
	}

	return migrator
}

func LoadSQLScript(path string) postgres.SQLScript {
	return func(db sql.ORM) error {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		err = db.Exec(string(content))
		return err
	}
}
