module grid

go 1.23.1

require (
	github.com/go-redis/redis/v8 v8.11.5
	github.com/segmentio/kafka-go v0.4.47
	server v0.0.0-00010101000000-000000000000
)

replace server => ../server

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/net v0.28.0 // indirect
)