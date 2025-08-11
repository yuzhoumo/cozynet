module mycelium

go 1.24.5

require (
	github.com/joho/godotenv v1.5.1
	github.com/mroth/weightedrand/v2 v2.1.0
	github.com/redis/go-redis/v9 v9.12.0
	golang.org/x/net v0.42.0
	google.golang.org/protobuf v1.36.7
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)

tool google.golang.org/protobuf/cmd/protoc-gen-go
