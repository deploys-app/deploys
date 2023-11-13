FROM golang:1.21.4

ENV CGO_ENABLED=0

WORKDIR /workspace

ADD go.mod go.sum ./
RUN go mod download
ADD . .
RUN go build -o .build/deploys -ldflags "-w -s" .

FROM debian:11-slim

RUN apt-get update && apt-get install -y \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/*

ENV PATH "$PATH:/app"

WORKDIR /app

COPY --from=0 --link /workspace/.build/* ./
ADD ./entrypoint.sh ./entrypoint.sh
ENTRYPOINT ["entrypoint.sh"]
