FROM golang:1.23-alpine as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o backend .

FROM scratch
COPY --from=builder /app/wesocket_service /app/wesocket_service
EXPOSE 8080
ENTRYPOINT ["/app/wesocket_service"]