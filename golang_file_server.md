 根目录URL处理
 
	package main
	
	import (
		"net/http"
	)
	
	func main() {
		http.Handle("/", http.FileServer(http.Dir("/tmp"))
		log.Fatal(http.ListenAndServe(":8080", nil))
		
		// or
		// log.Fatal(http.ListenAndServe(":8080", 
		//			http.FileServer(http.Dir("/usr/share/doc"))))
	}
		
备注：如果使用暴露dir的方式存在安全隐患的，也可以采用serveFile的方式进行

	http.HandleFunc("/static2/", func(w http.ResponseWriter, r *http.Request) {
		// !TODO Add check file access or other thing
   		http.ServeFile(w, r, r.URL.Path[1:])
	})

前缀目录url处理	

	package main
	
	import (
		"net/http"
	)
	
	func main() {
		http.Handle("/tmp/", http.StripPrefix("/tmp/", http.FileServer(http.Dir("/tmp/static"))))
		http.ListenAndServe(":8080", nil)
	}
	
参考：

1. [Golang. What to use? http.ServeFile(..) or http.FileServer(..)?](http://stackoverflow.com/questions/28793619/golang-what-to-use-http-servefile-or-http-fileserver)
2. [golang Http ServerFile](https://golang.org/pkg/net/http/#ServeFile)
3. [golang Http FileServer](https://golang.org/pkg/net/http/#FileServer)