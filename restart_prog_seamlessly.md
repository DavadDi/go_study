# 程序的无缝重启

![](http://www.do1618.com/wp-content/uploads/2017/01/restart_prog_seamlessly.png)

程序的无缝重启是对于 Client 透明的一种重启方案，即保证原有连接客户端的业务逻辑正常，同时启动一个新的进程用于处理后续的新连接的 Client 的处理。

无缝重启的核心点在于需要将原有服务端程序的 listenfd 复制到后启动的服务端程序中， Linux的 Fork 原语非常好的适合此种场景，因为Fork 采用 Copy-on-Write的技术，子进程直接共享了父进程的各种资源。

参考上图中的逻辑，parent process为已经运行的服务端程序，child process则为重启后接替parent process进程新进程。大体实现方式如下：
1. 首先定义重启信号或者标志位，触发重启流程。由于程序已经服务了 Client A...X, 所以其仍然要存在一段时间直至A....X的业务处理完毕；
2. 在重启流程中将父进程的 listenfd 通过Fork技术传递到 子进程中，由于设置了特殊的标志，子进程启动后能够明确自己的角色，在listenfd 上进行 accept 新连接的 client B....Y；
3. 待parent process进程处理完毕 Client A...X后，退出
4. child process的全局接管。


参考资料：

* [Nginx平滑升级和平滑重启](http://kgdbfmwfn.blog.51cto.com/5062471/1708258)
* [Graceful server restart with Go](https://blog.scalingo.com/2014/12/19/graceful-server-restart-with-go.html)   [Example](https://github.com/Scalingo/go-graceful-restart-example)
* [如何实现支持数亿用户的长连消息系统 | Golang高并发案例](http://chuansong.me/n/1641640)