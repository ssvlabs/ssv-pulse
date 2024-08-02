BINARY_DIR=${EXEC_DIR}/bin/benchmark
EXEC_DIR=cmd/benchmark
NODE_ADDR=beacon1=REPLACE_WITH_ADDR

.PHONY: default
default: build run

.PHONY: build
build:
	go build -o ${BINARY_DIR} ${EXEC_DIR}/main.go

.PHONY: run
run:
	./${BINARY_DIR} -addresses=${NODE_ADDR}

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