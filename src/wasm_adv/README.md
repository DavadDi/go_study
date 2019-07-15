# INTRO

> go version go1.12 darwin/amd64

## command 
```bash
$ GOOS=js GOARCH=wasm go build -o lib.wasm
# Reducing the size of Wasm files
$ gzip --best lib.wasm 
$ cp $(go env GOROOT)/misc/wasm/wasm_exec.js .
$ go run file_server.go
```

## In Chrome

* Type Addr: http://127.0.0.1:8080/wasm_exec.html

## see also

* [Go WebAssembly Tutorial - Building a Calculator Tutorial](https://tutorialedge.net/golang/go-webassembly-tutorial/)
* [Golang's syscall/js js.NewCallback is undefined](https://stackoverflow.com/questions/55800163/golangs-syscall-js-js-newcallback-is-undefined)


