FROM golang:1.25.7-alpine3.23

WORKDIR /workspace

ENV GOOS=linux
ENV GOARCH=arm
ENV CGO_ENABLED=0

ADD go.mod go.sum ./
RUN go mod download
ADD . .
RUN go build -o .build/pcl -ldflags "-w -s" ./cmd/pcl

FROM alpine:3.23

WORKDIR /app

COPY --from=0 /workspace/.build/ ./
ENTRYPOINT ["/app/pcl"]
