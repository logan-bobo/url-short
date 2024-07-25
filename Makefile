COMPOSE_TEST_FILE="${PWD}/docker-compose-test.yaml"

fmt:
	COMPOSE_FILE=${COMPOSE_TEST_FILE} docker compose run --rm --remove-orphans api-test go fmt
.PHONY:fmt

lint: fmt
	COMPOSE_FILE=${COMPOSE_TEST_FILE} docker compose run --rm --remove-orphans api-test golangci-lint run -v
.PHONY:lint

build:
	docker build . -t "url-short:latest"
.PHONY:build

build/test:
	docker build . -t "url-short:test" --target tester
.PHONY:build/test

run:
	docker compose up -d
.PHONY:run

stop:
	docker compose down
.PHONY:stop

test:
	COMPOSE_FILE=${COMPOSE_TEST_FILE} docker compose up -d
	COMPOSE_FILE=${COMPOSE_TEST_FILE} docker compose run --rm --remove-orphans api-test go test ./...
	COMPOSE_FILE=${COMPOSE_TEST_FILE} docker compose down
.PHONY:test

migrate: 
	goose -dir sql/schema postgres "postgres://url_short:password@localhost:5002/url_short" up 
.PHONY:migrate
