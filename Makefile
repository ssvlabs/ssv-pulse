EXEC_DIR=cmd/pulse
BINARY_DIR=${EXEC_DIR}/bin
BINARY_NAME=pulse
DOCKER_IMAGE_NANE=pulse
CONFIG_DIR=./configs
CONFIG_FILE=config.yaml
PORT=8080

GET_TOOL=env GOWORK=off go get -modfile=tool.mod -tool
RUN_TOOL=env GOWORK=off go tool -modfile=tool.mod

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
	go test -race -cover ./...

.PHONY: test_coverage
test_coverage:
	go test -race ./... -coverprofile=coverage.out

.PHONY: dep
dep:
	go mod download

.PHONY: vet
vet:
	go vet ./...

.PHONY: tools
# Keep tool.mod tools-only. Do not tidy it from the repo root, because that
# makes Go resolve application packages through tool.mod and pulls app deps in.
tools:
	${GET_TOOL} github.com/golangci/golangci-lint/v2/cmd/golangci-lint

#https://github.com/golangci/golangci-lint/blob/HEAD/.golangci.reference.yml
.PHONY: lint
lint:
	$(RUN_TOOL) github.com/golangci/golangci-lint/v2/cmd/golangci-lint run -v ./...

.PHONY: lint-fix
lint-fix:
	$(RUN_TOOL) github.com/golangci/golangci-lint/v2/cmd/golangci-lint run --fix -v ./...