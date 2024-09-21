FROM golang:1.23-alpine as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o grid .

FROM scratch
COPY --from=builder /app/grid /app/grid
EXPOSE 8083
ENTRYPOINT ["/app/grid"]