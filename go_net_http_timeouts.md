# net http timeouts

## 1. SetDeadline

[`net.Conn`](https://golang.org/pkg/net/#Conn) 具有相关函数 `Set[Read|Write]Deadline(time.Time)`, 参数是绝对时间，当时间超时后会将操作的 I/O 失败，并返回 timeout 的error。

Deadline并不是timeout超时，如果使用 `SetDeadline` 当做超时使用的话，需要在每一次的 `Read`和`Write`前调用来设置本次IO操作的到期时间。

我们可能不想使用 `SetDeadline`来设置超时，而是采用更高层次的超时函数通过 `net/http` 来随设置，但是在底层实现上仍然是通过底层调用 `SetDeadline`来实现的，读取或者写入数据不会重新设置超时的值。



## 2. Server Timeouts

![aa](http://www.do1618.com/wp-content/uploads/2018/02/Server_Timeouts.png)

对于 Http Server 来讲设置客户端连接的超时非常有必要，以用来规避大量连接的静默客户端或者异常退出客户端消耗掉服务端大量的文件句柄，甚至导致Accept报错。

```
http: Accept error: accept tcp [::]:80: accept4: too many open files; retrying in 5ms
```



HttpServer 暴露了两个超时设置 `ReadTimeout` 和 `WriteTimeout`,  [server.go](https://tip.golang.org/src/net/http/server.go?s=72734:76093#L2389)

```go
type Server struct {
    // ReadTimeout is the maximum duration for reading the entire
    // request, including the body.
    //
    // Because ReadTimeout does not let Handlers make per-request
    // decisions on each request body's acceptable deadline or
    // upload rate, most users will prefer to use
    // ReadHeaderTimeout. It is valid to use them both.
    ReadTimeout time.Duration

    // ReadHeaderTimeout is the amount of time allowed to read
    // request headers. The connection's read deadline is reset
    // after reading the headers and the Handler can decide what
    // is considered too slow for the body.
    ReadHeaderTimeout time.Duration

    // WriteTimeout is the maximum duration before timing out
    // writes of the response. It is reset whenever a new
    // request's header is read. Like ReadTimeout, it does not
    // let Handlers make decisions on a per-request basis.
    WriteTimeout time.Duration

    // IdleTimeout is the maximum amount of time to wait for the
    // next request when keep-alives are enabled. If IdleTimeout
    // is zero, the value of ReadTimeout is used. If both are
    // zero, ReadHeaderTimeout is used.
    IdleTimeout time.Duration
    
    // ...
}

srv := &http.Server{
    ReadTimeout: 5 * time.Second,
    WriteTimeout: 10 * time.Second,
}
log.Println(srv.ListenAndServe())
```



`ReadTimeout` 超时时间涵盖了 Accept 到 Read Header和Body的时间，具体参见上图；其实现方式是 `net/http` 在 Accept 后马上调用 `SetReadDeadline`实现的;

`WriteTimeout` 一般包括 Header 读取后到写入 Response 结束，参见上图所示；

[源码](https://github.com/golang/go/blob/3ba31558d1bca8ae6d2f03209b4cae55381175b3/src/net/http/server.go#L750)如下：



```go
func (c *conn) readRequest(ctx context.Context) (w *response, err error) {
	if c.hijacked() {
		return nil, ErrHijacked
	}

	if d := c.server.ReadTimeout; d != 0 {
		c.rwc.SetReadDeadline(time.Now().Add(d))
	}
	if d := c.server.WriteTimeout; d != 0 {
		defer func() {
			c.rwc.SetWriteDeadline(time.Now().Add(d))
		}()
	}
	
	// ...
}
```

但是涉及到 HTTPS，由于中间多出了 TLS 握手过程，因此情况更加复杂，`SetWriteDeadline`会在 Accept后立即设置以包括在 TLS 握手过程中涉及的 写入操作，在这种情况下会导致 `WriteTimeout` 的时间从连接开始到 HEAD 被读取，参见上图中的虚线部分，[源码参见](https://github.com/golang/go/blob/3ba31558d1bca8ae6d2f03209b4cae55381175b3/src/net/http/server.go#L1477-L1483)：

```go
// Serve a new connection.
func (c *conn) serve(ctx context.Context) {
	c.remoteAddr = c.rwc.RemoteAddr().String()
	// ...

	if tlsConn, ok := c.rwc.(*tls.Conn); ok {
		if d := c.server.ReadTimeout; d != 0 {
			c.rwc.SetReadDeadline(time.Now().Add(d))
		}
		if d := c.server.WriteTimeout; d != 0 {
			c.rwc.SetWriteDeadline(time.Now().Add(d))
		}
	// ...
	}
}
```



通过以上的分析可以得知， `http.Server` 中的 `http.ListenAndServe`/`http.ListenAndServeTLS`/`http.Serve`都默认没有提供设置超时的设置，这些函数默认都关闭了超时机制，而且也没有提供明确的函数启动超时机制，这样会检查导致服务端出现文件句柄泄露，最终拖垮服务程序，因此需要通过设置 `http.Server`实例中的 ReadTimeout 和 WriteTimeout 属性值来设置。

### About streaming

`ServeHTTP` 从不能获取到底层的 `net.Conn` 因此在涉及到需要流式发送 Response 而必须清除掉 `WriteTimeout`的时候就没有办法做到，则可能也是为什么默认值是0的原因。因为不能够获取到底层的  `net.Conn` 对象，因此没有办法在每次 `Write`之前调用 `SetWriteDeadline`来获取到特定的超时设置。



同样，也没有办法取消一个阻塞的 `ResponseWriter.Write`，因为 `ResponseWriter.Close` 也不能简单唤醒一个阻塞的写操作，因此也没有办法手工设置定时器来解决这个问题。



## 3. Client Timeout

![](http://www.do1618.com/wp-content/uploads/2018/02/Client_Timeouts.png) 

Client 侧的超时可能更加简单或者复杂，取决于我们的具体使用场景。



最简单的方式是采用以下方式：

```go
c := &http.Client{
    Timeout: 15 * time.Second,
}
resp, err := c.Get("https://xxx.yyyy/")
```

同服务端的情况以下，如果使用 `http.Get`而不设置超时时间，可能会导致程序卡在某个地方不能自拔。`http.Get` 提供了更加细粒度的控制：

- `net.Dialer.Timeout` limits the time spent establishing a TCP connection (if a new one is needed).
- `http.Transport.TLSHandshakeTimeout` limits the time spent performing the TLS handshake.
- `http.Transport.ResponseHeaderTimeout` limits the time spent reading the headers of the response.
- `http.Transport.ExpectContinueTimeout` limits the time the client will wait between sending the request headers *when including an Expect: 100-continue*and receiving the go-ahead to send the body. Note that setting this in 1.6 [will disable HTTP/2](https://github.com/golang/go/issues/14391) (`DefaultTransport` [is special-cased from 1.6.2](https://github.com/golang/go/commit/406752b640fcc56a9287b8454564cffe2f0021c1#diff-6951e7593bfb1e773c9121df44df1c36R179)).

使用如下：

```go
c := &http.Client{
    Transport: &http.Transport{
        Dial: (&net.Dialer{
                Timeout:   30 * time.Second,
                KeepAlive: 30 * time.Second,
        }).Dial,
        TLSHandshakeTimeout:   10 * time.Second,
        ResponseHeaderTimeout: 10 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
    }
}
```

没有特别具体的时间来确定发送请求的发送所花费的时间。花费在读取影响的主题数据可以通过 `time.Timer`来进行控制，因为这个行为发生在Client的方法返回以后。 最后，在go 1.7 中，定义了一个 `http.Transport.IdleConnTimeout`，这个不是控制 client 请求的阻塞阶段，而是表明连接在连接池中空闲的时间。注意 Client 默认会追溯重定向的地址，`http.Client.Timeout`这个时间也包括重定向获取的消耗，而 `http.Transport` 是更底层的控制，不会涉及到重定向的相关时间消耗。



### Cancel and Context

`net/http`提供了两种方式来取消客户端的请求：`Request.Cancel` 和 `Context` (1.7)，go推荐使用 `Context`, `Request.Cancel`会逐渐被废弃。

**Request.Cancel**

```go
package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func main() {
	c := make(chan struct{})
	timer := time.AfterFunc(5*time.Second, func() {
		close(c)
	})

        // Serve 256 bytes every second.
	req, err := http.NewRequest("GET", "http://httpbin.org/range/2048?duration=8&chunk_size=256", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Cancel = c

	log.Println("Sending request...")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	log.Println("Reading body...")
	for {
		timer.Reset(2 * time.Second)
                // Try instead: timer.Reset(50 * time.Millisecond)
		_, err = io.CopyN(ioutil.Discard, resp.Body, 256)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
	}
}
```



**context.WithCancel**

```go
ctx, cancel := context.WithCancel(context.TODO())
timer := time.AfterFunc(5*time.Second, func() {
	cancel()
})

req, err := http.NewRequest("GET", "http://httpbin.org/range/2048?duration=8&chunk_size=256", nil)
if err != nil {
	log.Fatal(err)
}
req = req.WithContext(ctx)
```



## 4. 采用 httptrace 跟踪细节

[TODO]

## 参考

1. [The complete guide to Go net/http timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/)
2. [So you want to expose Go on the Internet](https://blog.cloudflare.com/exposing-go-on-the-internet/)
3. [How to correctly use context.Context in Go 1.7](https://medium.com/@cep21/how-to-correctly-use-context-context-in-go-1-7-8f2c0fafdf39)
4. [Go Concurrency Patterns: Context](https://blog.golang.org/context)

