# Go 初始化过程

## 1. go程序初始化流程

* 1. 首先是import pkg的初始化过程
* 2. pkg中定义的const变量初始化
* 3. pkg中定义的var全局变量
* 4. pkg中定义的init函数，可能有多个

![int_seq](http://www.do1618.com/wp-content/uploads/2017/05/go_init_seq.png)

From: [Programming in Go]

在 pkg 内，pkg level的 var 变量按照声明的顺序进行声明，但是要在依赖初始化变量的后面。

初始化的顺序： d, b, c, a

```go
var (
	a = c + b
	b = f()
	c = f()
	d = 3
)

func f() int {
	d++
	return d
}
```

Ref [Program_initialization_and_execution](https://golang.org/ref/spec#Program_initialization_and_execution)

## 2. go test程序的初始化

有时候运行 go test我们期望能够在测试前进行 setup 操作，而在测试完成后执行 teardown 操作，但是由于 go test 运行的不确定性，因此 go test 框架提供了一个 Main 函数实现这一机制。

```go
	func TestMain(m *testing.M) {
		// call flag.Parse() here if TestMain uses flags
		os.Exit(m.Run())
	}
```