package migrations

import (
	"os"

	"codebase/models"

	"persistent/sql/sql"
	"tools/migrations/postgres"
)

var (
	scripts    = map[string]postgres.SQLMigration{}
	migrations = []string{
		"20250806084945_create_payment_table",
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
