BINARY_DIR=${EXEC_DIR}/bin/benchmark
EXEC_DIR=cmd
DOCKER_IMAGE_NANE=benchmark
NODE_ADDR=REPLACE_WITH_ADDR
NETWORK=REPLACE_WITH_NETWORK_NAME

.PHONY: default
default: build run

.PHONY: build
build:
	go build -o ${BINARY_DIR} ${EXEC_DIR}/main.go

.PHONY: run
run:
	./${BINARY_DIR} -addresses=${NODE_ADDR} -network=${NETWORK}

########## DOCKER
.PHONY: docker-build
docker-build:
	docker build -t ${DOCKER_IMAGE_NANE} -f build/Dockerfile .

.PHONY: docker-run
docker-run:
	docker run ${DOCKER_IMAGE_NANE} -addresses=${NODE_ADDR} -network=${NETWORK}
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