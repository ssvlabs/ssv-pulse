BINARY_DIR=${EXEC_DIR}/bin/benchmark
EXEC_DIR=cmd/benchmark
DOCKER_IMAGE_NANE=benchmark
NODE_ADDR=REPLACE_WITH_ADDR
NETWORK=REPLACE_WITH_NETWORK_NAME
LOG_FILE_PATH=REPLACE_WITH_PATH

.PHONY: build
build:
	go build -o ${BINARY_DIR} ${EXEC_DIR}/main.go

.PHONY: run-benchmark
run-benchmark: build
	./${BINARY_DIR} benchmark --address=${NODE_ADDR} --network=${NETWORK}

.PHONY: run-analyzer
run-analyzer: build
	./${BINARY_DIR} log-analyzer --logFilePath=${LOG_FILE_PATH}

########## DOCKER
.PHONY: docker-build
docker-build:
	docker build -t ${DOCKER_IMAGE_NANE} -f build/Dockerfile .

.PHONY: docker-run
docker-run:
	docker run ${DOCKER_IMAGE_NANE} -address=${NODE_ADDR} -network=${NETWORK}
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