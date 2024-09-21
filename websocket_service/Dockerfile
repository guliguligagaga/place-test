FROM golang:1.23-alpine as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o websocket_service .

FROM scratch
COPY --from=builder /app/websocket_service /app/websocket_service
EXPOSE 8082
ENTRYPOINT ["/app/websocket_service"]