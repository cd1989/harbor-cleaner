FROM alpine:3.8

LABEL maintainer="xxyydream@gmail.com"

WORKDIR /workspace

COPY ./bin/cleaner /workspace/cleaner
COPY ./cmd/config.yaml /workspace/config.yaml

ENTRYPOINT ["./cleaner"]
