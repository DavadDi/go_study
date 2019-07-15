all:
	GOOS=js GOARCH=wasm go build -o lib.wasm lib.go
	gzip --best lib.wasm
	go run file_server.go
