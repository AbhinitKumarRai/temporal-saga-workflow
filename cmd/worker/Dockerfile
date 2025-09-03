FROM golang:1.21 as builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /worker ./cmd/worker

FROM gcr.io/distroless/base-debian12
COPY --from=builder /worker /worker
ENTRYPOINT ["/worker"]

