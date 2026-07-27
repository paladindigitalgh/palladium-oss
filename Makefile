.PHONY: build run test test-integration vet fmt tidy clean \
	migrate-up migrate-status db-up db-down

build:
	go build -o bin/palladium-server ./cmd/server
	go build -o bin/palladium-migrate ./cmd/migrate

run:
	go run ./cmd/server

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

tidy:
	go mod tidy

clean:
	rm -rf bin

migrate-up:
	go run ./cmd/migrate up

migrate-status:
	go run ./cmd/migrate status

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
