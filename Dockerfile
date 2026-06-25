FROM golang:1.26.4-trixie

ENV CGO_ENABLED=0

WORKDIR /workspace

# VERSION stamps the binary so `deploys check-update` can report it; the release
# workflow passes the git tag. Defaults to "dev" for a plain `docker build`.
ARG VERSION=dev

ADD go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
ADD . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o .build/deploys -ldflags "-w -s -X main.version=${VERSION}" .

FROM debian:13-slim

RUN apt-get update && apt-get install -y \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/*

ENV PATH "$PATH:/app"

WORKDIR /app

COPY --from=0 --link /workspace/.build/* ./
ADD ./entrypoint.sh ./entrypoint.sh
ENTRYPOINT ["entrypoint.sh"]
