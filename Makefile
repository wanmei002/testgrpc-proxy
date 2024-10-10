BUILD_DEST_DIR ?= build
TARGET ?= testgrpc

PROTO_SRCS = $(shell find proto -type f -name '*.proto')

.PHONY: build
build:
	mkdir -p ${BUILD_DEST_DIR}
	@echo "building ${BUILD_DEST_DIR}/${TARGET} ..."
	go env -w GO111MODULE=on
	go env -w GOPROXY=https://goproxy.cn,direct
	CGO_ENABLED=0 go build -o ${BUILD_DEST_DIR}/${TARGET} main.go

.PHONY: docker
docker:
	@echo "build docker image wanmei002/${TARGET}:devel2"
	docker buildx build -f Dockerfile --build-arg TARGET=${TARGET} --build-arg APP_VERSION=${APP_VERSION} --sbom false --provenance false \
		-t wanmei002/${TARGET}:devel2 --platform=linux/amd64 . --push



.PHONY: proto
proto: ${PROTO_SRCS}
	buf generate

