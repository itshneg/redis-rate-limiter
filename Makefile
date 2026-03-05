docker:
	docker compose up -d

test:
	go test ./... -v -race -coverprofile=coverage.out

test-docker: docker test

lint:
	golangci-lint run

coverage:
	go test ./... -coverprofile=coverage.out

clean:
	rm -f coverage.out

all: test lint coverage