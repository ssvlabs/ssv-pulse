EXEC_DIR=cmd/benchmark
BINARY_DIR=${EXEC_DIR}/bin
BINARY_NAME=benchmark
DOCKER_IMAGE_NANE=benchmark
CONFIG_DIR=./configs
CONFIG_FILE=config.yaml

.PHONY: build
build:
	go build -o ${BINARY_DIR}/${BINARY_NAME} ${EXEC_DIR}/main.go
	@cp $(CONFIG_DIR)/$(CONFIG_FILE) $(BINARY_DIR)/

.PHONY: run-benchmark
run-benchmark: build
	@cd ${BINARY_DIR} && ./${BINARY_NAME} benchmark

.PHONY: run-analyzer
run-analyzer: build
	@cd ${BINARY_DIR} && ./${BINARY_NAME} log-analyzer

########## DOCKER
.PHONY: docker-build
docker-build:
	docker build -t ${DOCKER_IMAGE_NANE} -f build/Dockerfile .
##########

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