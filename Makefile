-include .env
export

dev:
	goreload -d ./cmd/pcl/

test-schema:
	go test -v ./schema

test-unit:
	go test ./internal/...

test-unit-cover:
	go test -cover ./internal/...

test-unit-coverage:
	go test -coverprofile=coverage.out ./internal/... && go tool cover -func=coverage.out && go tool cover -html=coverage.out

test-integration:
	go test -v ./tests
