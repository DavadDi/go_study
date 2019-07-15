all:
	GOOS=js GOARCH=wasm go build -o lib.wasm lib.go
	go run file_server.go
