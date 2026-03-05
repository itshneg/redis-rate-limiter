docker:
	docker compose up -d

test:
	docker compose up -d
	go test ./... -v -race -coverprofile=coverage.out
	docker compose down

lint:
	golangci-lint run

coverage:
	go test ./... -coverprofile=coverage.out

clean:
	rm -f coverage.out

all: test lint coverage