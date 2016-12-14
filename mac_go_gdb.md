# 使用GDB调试golang

## 1 Mac gdb准备

参考：

1. [Codesign gdb on Mac OS X Yosemite (10.10.2)](http://andresabino.com/2015/04/14/codesign-gdb-on-mac-os-x-yosemite-10-10-2/)
2. [How to get gdb installed with homebrew code signed](http://stackoverflow.com/questions/18423124/please-check-gdb-is-codesigned-see-taskgated8-how-to-get-gdb-installed-w)

### 1.1 安装gdb

安装：
	
	$ brew tap homebrew/dupes
	$ brew install gdb
	
	安装目录 /usr/local/bin/gdb

查看版本：

	$gdb -v
	GNU gdb (GDB) 7.11   # GDB 版本必须大于 7.1	
如果使用gdb直接调试程序会报以下错误：

	$ gdb a.out
	Starting program: a.out
    Unable to find Mach task port for process-id XXXXX: (os/kern) failure (0x5).
     (please check gdb is codesigned - see taskgated(8))
	Start Keychain Access application (/Applications/Utilities/Keychain Access.app)
	

出错的原因是因为mac下，访问其他的进程需要gdb进行数字签名。需要生成一个*签名证书*且注册到*System*中。
	
### 1.2 生成证书和签名gdb

Finder -> Go -> Utilities -> Keychain Access -> Certificate Assistant -> Create a Certificate ...

**Certtificat Type: Code Signing**    
![gdb_crt](http://www.do1618.com/wp-content/uploads/2016/12/gdb_crt.png)

一路向下默认值，但是证书注册的位置需要选择 **System**    
![key_chain](http://www.do1618.com/wp-content/uploads/2016/12/gdb_crt_system.png)	

添加成功后，双击证书， 在Turst section中选择 Always Trust。

然后对于gdb使用证书签名

	$ sudo killall taskgated
	$ codesign -fs gdb-cert /usr/local/bin/gdb

## 2. 使用gdb调试

[Debugging Go Code with GDB(官方文档)](https://golang.org/doc/gdb)

### 2.1 编译程序

发布版本删除调试符号
	
	go build -ldflags “-s -w”

方便调试关闭内联优化

	go build -gcflags “-N -l”
	

### 2.2 调试命令

	$gdb regexp.test -d $GOROOT
	(gdb) source ~/go/src/runtime/runtime-gdb.py

	(gdb) list
	(gdb) list line
	(gdb) list file.go:line
	(gdb) break line
	(gdb) break file.go:line
	(gdb) disas
	
	(gdb) bt
	(gdb) frame n
	
	(gdb) info locals
	(gdb) info args
	(gdb) p variable
	(gdb) whatis variable	
	
	(gdb) info variables regexp
	
	(gdb) x/15xb 0x42121240
	
Go Extensions

	(gdb) p var
	(gdb) p $len(var)
	
	(gdb) p $dtype(var)
	(gdb) iface var
	
	(gdb) info goroutines
	(gdb) goroutine n cmd
	(gdb) help goroutine

### 2.3 其他事项

	gdb 对于 interface{} 类型不能够提供反解析。
	

### 3. godebug

跨平台的go debug的工具

### 3.1 安装
	
	go get github.com/mailgun/godebug
	
### 3.2 测试

在需要调试的代码中添加一下代码：
	
	_ = "breakpoint"
	
启动测试：
	$ godebug run -instrument= gofiles... [arguments...]
	
	例如：
	$ godebug run -instrument=github.com/astaxie/beego/orm main.go
	

### 3.3  Debugger commands:

The current commands are:

command              | result
---------------------|------------------------
h(elp)               | show help message
n(ext)               | run the next line
s(tep)               | run for one step
c(ontinue)           | run until the next breakpoint
l(ist)               | show the current line in context of the code around it
p(rint) [expression] | print a variable or any other Go expression
q(uit)               | exit the program

### 3.4 总结

godebug还是一个比较新的项目，能扩张的命令也比较少，但是对于一般的调试还是非常方便，能够很好支持golang的各种类型，例如 ** interface{} **
	

参考：

1. [Using the gdb debugger with Go](https://blog.codeship.com/using-gdb-debugger-with-go/)
2. [GDB调试Go程序](http://blog.studygolang.com/2012/12/gdb%E8%B0%83%E8%AF%95go%E7%A8%8B%E5%BA%8F/)
2. [Go Data Structures: Interfaces](https://research.swtch.com/interfaces)
3. [golang interface analysis by gdb](http://compasses.github.io/2015/10/23/golang-interface-analysis-by-gdb/)
4. [Debugging Go (golang) programs with gdb](http://thornydev.blogspot.com/2014/01/debugging-go-golang-programs-with-gdb.html)
5. [Go 程序调试工具 godebug](http://studygolang.com/p/godebug) [godebug](https://github.com/mailgun/godebug)