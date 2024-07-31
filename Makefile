BINARY_PATH=bin/benchmark

.PHONY: default
default: build run

.PHONY: build
build:
	go build -o ${BINARY_PATH} main.go

.PHONY: run
run:
	./${BINARY_PATH}

.PHONY: clean
clean:
	go clean

.PHONY: test
test:
	go test ./...

.PHONY: test_coverage
test_coverage:
	go test ./... -coverprofile=coverage.out

.PHONY: dep
dep:
	go mod download

.PHONY: vet
vet:
	go vet ./...