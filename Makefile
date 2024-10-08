.PHONY: gen clean server client test cert

gen:
    protoc --proto_path=proto --go_out=. --go-grpc_out=. proto/*.proto

clean:
    rm pb/*.go

server:
    go run cmd/server/main.go -port=8080

client:
    go run cmd/client/main.go -address 0.0.0.0:8080

test:
    go test -v -cover ./...

cert:
    cd cert && ./gen.sh
