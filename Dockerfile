FROM golang:1.22 as builder
LABEL maintainer=expvent@expvent.com

WORKDIR /workspace
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked --mount=type=cache,target=/root/.cache,sharing=locked \
    make -e BUILD_DEST_DIR=build build

FROM ubuntu:20.04
LABEL maintainer=expvent@expvent.com

ARG TARGET
WORKDIR /
ENV TARGET=${TARGET}
USER root
COPY --from=builder /workspace/build/* ./

RUN chmod +x /${TARGET}

ENTRYPOINT /${TARGET}


