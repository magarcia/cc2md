binary := "cc2md"
VERSION := "0.1.0"
COMMIT := `git rev-parse --short HEAD`
DATE := `date -u +%Y-%m-%d`
LDFLAGS := "-s -w -X main.version=" + VERSION + " -X main.commit=" + COMMIT + " -X main.date=" + DATE

build:
    go build -ldflags "{{LDFLAGS}}" -o {{binary}} .

test:
    go test ./...

test-verbose:
    go test -v ./...

install:
    go install -ldflags "{{LDFLAGS}}" .

clean:
    rm -f {{binary}} {{binary}}-*

build-all:
    GOOS=darwin GOARCH=arm64 go build -ldflags "{{LDFLAGS}}" -o {{binary}}-darwin-arm64 .
    GOOS=darwin GOARCH=amd64 go build -ldflags "{{LDFLAGS}}" -o {{binary}}-darwin-amd64 .
    GOOS=linux GOARCH=amd64 go build -ldflags "{{LDFLAGS}}" -o {{binary}}-linux-amd64 .
    GOOS=linux GOARCH=arm64 go build -ldflags "{{LDFLAGS}}" -o {{binary}}-linux-arm64 .

run *ARGS:
    go run -ldflags "{{LDFLAGS}}" . {{ARGS}}

fmt:
    gofmt -w .

vet:
    go vet ./...

lint:
    golangci-lint run ./...

check:
    test -z "$(gofmt -l .)" && go vet ./... && go test ./... && go build ./...
