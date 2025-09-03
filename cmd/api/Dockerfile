FROM golang:1.21 as builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /api ./cmd/api

FROM gcr.io/distroless/base-debian12
COPY --from=builder /api /api
ENTRYPOINT ["/api"]

