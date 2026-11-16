.PHONY: run test test-integration lint build done

run:
	go run . run

test:
	go test ./... -race -cover

# Needs a local Postgres; wipes DB_NAME (must end in _test). Schema: $SCHEMA_DIR,
# default ../storybook_management_app/migrations.
test-integration:
	DB_NAME=$${DB_NAME:-storybook_test} go test -tags integration -race -count=1 ./handler/

lint:
	go vet ./... && go vet -tags integration ./handler/

build:
	go build -o bin/app .

done: lint test build
