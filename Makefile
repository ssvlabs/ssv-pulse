EXEC_DIR=cmd/pulse
BINARY_DIR=${EXEC_DIR}/bin
BINARY_NAME=pulse
DOCKER_IMAGE_NANE=pulse
CONFIG_DIR=./configs
CONFIG_FILE=config.yaml
PORT=8080

.PHONY: build
build:
	go build -o ${BINARY_DIR}/${BINARY_NAME} ${EXEC_DIR}/main.go
	@cp $(CONFIG_DIR)/$(CONFIG_FILE) $(BINARY_DIR)/

.PHONY: run-benchmark
run-benchmark: build
	@cd ${BINARY_DIR} && ./${BINARY_NAME} benchmark

.PHONY: run-analyzer
run-analyzer: build
	@cd ${BINARY_DIR} && ./${BINARY_NAME} analyzer

########## DOCKER
.PHONY: docker-build
docker-build:
	docker build -t ${DOCKER_IMAGE_NANE} -f build/Dockerfile .

.PHONY: docker-run-benchmark
docker-run-benchmark:
	docker run -p ${PORT}:${PORT} ${DOCKER_IMAGE_NANE} benchmark --port=${PORT}

.PHONY: docker-compose-up
docker-compose-up:
	docker-compose -f ./build/docker-compose.yml up -d

.PHONY: docker-compose-down
docker-compose-down:
	docker-compose -f ./build/docker-compose.yml down
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

#https://github.com/golangci/golangci-lint/blob/HEAD/.golangci.reference.yml
.PHONY: lint
lint:
	golangci-lint run -v ./...

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix -v ./...