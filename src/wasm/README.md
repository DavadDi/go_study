# INTRO

## command 
```bash
$ GOOS=js GOARCH=wasm go build -o test.wasm

$ cp $(go env GOROOT)/misc/wasm/wasm_exec.{html,js} .
$ go get -u github.com/shurcooL/goexec
$ goexec 'http.ListenAndServe(":8080", http.FileServer(http.Dir(".")))'
```



### In Chrome

1. open "http://localhost:8080/wasm_exec.html"
2. click button ”Run“
3. Press Right Mouse Button, "Inspect" 
4. Choose Console , Can see "Hello, WebAssembly!" 



## see also

* [WebAssembly](https://github.com/golang/go/wiki/WebAssembly)
* [Learn Web Assembly the hard way](https://agniva.me/wasm/2018/05/17/wasm-hard-way.html)
* [Go and wasm: generating and executing wasm with Go](https://blog.gopheracademy.com/advent-2017/go-wasm/)
* [WebAssembly 现状与实战](https://www.ibm.com/developerworks/cn/web/wa-lo-webassembly-status-and-reality/index.html)
* [Tiny)Go to WebAssembly](https://dev.to/sendilkumarn/tiny-go-to-webassembly-5168)
* [如何使用 WebAssembly 提升性能](https://www.infoq.cn/article/2IHWa2Ivbvw*hFw6fvk6)
* [WebAssembly 对比 JavaScript 及其使用场景](https://github.com/Troland/how-javascript-works/blob/master/webassembly.md#toc2)
* [Go WebAssembly Tutorial - Building a Calculator Tutorial](https://tutorialedge.net/golang/go-webassembly-tutorial/)
* [Johan Brandhorst - Get Going with WebAssembly-Video](https://www.youtube.com/watch?v=iTrx0BbUXI4)
* [Practice your Go WebAssembly with a Game](https://medium.com/@didil/practice-your-go-webassembly-with-a-game-7195dabbfc44)


