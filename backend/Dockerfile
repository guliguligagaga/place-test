FROM golang:1.23-alpine as builder
WORKDIR /app
ARG SERVICE_NAME

COPY . .

RUN go build -o /bin/app /backend/main/${SERVICE_NAME}

FROM gcr.io/distroless/static
COPY --from=builder /bin/app /app
EXPOSE 8080
ENTRYPOINT ["/app"]