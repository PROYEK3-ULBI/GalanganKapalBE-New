.PHONY: run build tidy test fmt lint hashpw

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

tidy:
	go mod tidy

test:
	go test ./...

fmt:
	gofmt -s -w .

# Generate a bcrypt hash. Usage: make hashpw PW=mypassword
hashpw:
	@go run ./cmd/hashpw $(PW)
