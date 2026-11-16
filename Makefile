.PHONY: run migrate test test-integration lint build done

run:
	go run . run

migrate:
	go run . migrate up

test:
	go test ./... -race -cover

# Needs a local Postgres; wipes DB_NAME (must end in _test).
test-integration:
	DB_NAME=$${DB_NAME:-storybook_test} go test -tags integration -race -count=1 ./handler/

lint:
	go vet ./... && go vet -tags integration ./handler/

build:
	go build -o bin/app .

done: lint test build
