FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o service-courier ./cmd/service-courier
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o service-courier-worker ./cmd/worker


FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/service-courier /service-courier
COPY --from=builder /app/service-courier-worker /service-courier-worker
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/service-courier"]