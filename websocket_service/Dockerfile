FROM rust:1.68 as builder
WORKDIR /src/websocket
COPY . .
RUN cargo install --path .

FROM debian:buster-slim
RUN apt-get update && apt-get install -y libssl-dev && rm -rf /var/lib/apt/lists/*
COPY --from=builder /usr/local/cargo/bin/websocket /usr/local/bin/websocket
CMD ["websocket"]