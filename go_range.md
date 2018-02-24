

# Go Range Loop Internals

## 1. 问题：这段程序会退出吗？

Dave's tweet: [#golang](https://twitter.com/hashtag/golang?src=hash) pop quiz: does this program terminate?

```go
package main

import "fmt"

func main() {
	v := []int{1, 2, 3}
	for i := range v {
		v = append(v, i)
	}

	fmt.Println("The End")
}
```



## 2. 关于 Range 

go语言中，for标准的用法如下：

```go
for i := 0; i < 3; i++ { 
    // ....
}
```



`range` 是基于关键字 `for`做了一层语法糖而已，常见用法如下：

```go
for i := range v {
	// ...
}
```

`range`支持的类型如下：

| type    | syntactic sugar for                                      |
| ------- | -------------------------------------------------------- |
| array   | the array                                                |
| string  | struct holding len + a pointer to the backing array      |
| slice   | struct holding len, cap + a pointer to the backing array |
| map     | pointer to a struct                                      |
| channel | pointer to a struct                                      |

> 在 go 语言中有一个非常重要的约定：Everything is pass by value.  当然 map、slice、channel 等底层都是基于指针的结构，因此 pass by value也只是将顶层对象复制了一份，底层的数据仍然指向同一份，有点类似实现了 pointer 的效果。

无论是 `for` 还是 `range`其定义的局部变量都会被复用。

例如以下代码：

```go
names := []string{"a", "b"}    
for _, name := range names {
     fmt.Printf("Addr: %p\n", &name)
     fmt.Println("")
}
```

在实际运行结果中，我们可以看到无论names的大小，打印出来的地址始终为同一个地址。因为变量复用那么采用以下方式使用，则不能够正常工作：

```go
names := []string{"a", "b"}   
newNames := []*string{}

for _, name := range names {
     fmt.Printf("Addr: %p\n", &name)
     newNames = appends(newNames, &name)  // 由于 name 变量被复用，因此newNames 中都为同一个变量
     fmt.Println("")
}
```



## 3. Range 内部原理

在go gcc版本的编译器源码中，Range [注释如下](https://github.com/golang/gofrontend/blob/master/go/statements.cc#L5358)

```go
  // Arrange to do a loop appropriate for the type.  We will produce
  //   for INIT ; COND ; POST {
  //           ITER_INIT
  //           INDEX = INDEX_TEMP
  //           VALUE = VALUE_TEMP // If there is a value
  //           original statements
  //   }
  
    if (range_type->is_slice_type())
    this->lower_range_slice(...);
  else if (range_type->array_type() != NULL)
    this->lower_range_array(...);
  else if (range_type->is_string_type())
    this->lower_range_string(...);
  else if (range_type->map_type() != NULL)
    this->lower_range_map(...);
  else if (range_type->channel_type() != NULL)
    this->lower_range_channel(...;
  else
    go_unreachable();
```

[lower_range_array](https://github.com/golang/gofrontend/blob/master/go/statements.cc#L5458)

```go
void For_range_statement::lower_range_array(...)
{
  // The loop we generate:
  //   len_temp := len(range)
  //   range_temp := range
  //   for index_temp = 0; index_temp < len_temp; index_temp++ {
  //           value_temp = range_temp[index_temp]
  //           index = index_temp
  //           value = value_temp
  //           original body
  //   }

  // Set *PINIT to
  //   var len_temp int
  //   len_temp = len(range)
  //   index_temp = 0
  ...
}
```



[lower_range_slice()](https://github.com/golang/gofrontend/blob/master/go/statements.cc#L5563)

```go
void For_range_statement::lower_range_slice(...)
{
  // The loop we generate:
  //   for_temp := range
  //   len_temp := len(for_temp)
  //   for index_temp = 0; index_temp < len_temp; index_temp++ {
  //           value_temp = for_temp[index_temp]
  //           index = index_temp
  //           value = value_temp
  //           original body
  //   }
  //
  // Using for_temp means that we don't need to check bounds when
  // fetching range_temp[index_temp].

  // Set *PINIT to
  //   range_temp := range
  //   var len_temp int
  //   len_temp = len(range_temp)
  //   index_temp = 0    
    ....
}
```



[lower_range_string](https://github.com/golang/gofrontend/blob/master/go/statements.cc#L5661:22)

```go
void For_range_statement::lower_range_string(...)
{
  // The loop we generate:
  //   len_temp := len(range)
  //   var next_index_temp int
  //   for index_temp = 0; index_temp < len_temp; index_temp = next_index_temp {
  //           value_temp = rune(range[index_temp])
  //           if value_temp < utf8.RuneSelf {
  //                   next_index_temp = index_temp + 1
  //           } else {
  //                   value_temp, next_index_temp = decoderune(range, index_temp)
  //           }
  //           index = index_temp
  //           value = value_temp
  //           // original body
  //   }

  // Set *PINIT to
  //   len_temp := len(range)
  //   var next_index_temp int
  //   index_temp = 0
  //   var value_temp rune // if value_temp not passed in
  ...
}
```



[lower_range_map](https://github.com/golang/gofrontend/blob/master/go/statements.cc#L5809)

```go
 void For_range_statement::lower_range_map(...)
 {
  // The runtime uses a struct to handle ranges over a map.  The
  // struct is built by Map_type::hiter_type for a specific map type.

  // The loop we generate:
  //   var hiter map_iteration_struct
  //   for mapiterinit(type, range, &hiter); hiter.key != nil; mapiternext(&hiter) {
  //           index_temp = *hiter.key
  //           value_temp = *hiter.val
  //           index = index_temp
  //           value = value_temp
  //           original body
  //   }

  // Set *PINIT to
  //   var hiter map_iteration_struct
  //   runtime.mapiterinit(type, range, &hiter)
  ...
 }
```



[lower_range_channel](https://github.com/golang/gofrontend/blob/master/go/statements.cc#L5911)

```go
void For_range_statement::lower_range_channel(...)
{
  // The loop we generate:
  //   for {
  //           index_temp, ok_temp = <-range
  //           if !ok_temp {
  //                   break
  //           }
  //           index = index_temp
  //           original body
  //   }

  // We have no initialization code, no condition, and no post code.   
    ...
}
```



通过以上生成源码的代码注释可以看出：

1. Array/Slice/String 对象的 Range 生成方式基本上类似，都会在初始化的时候 Copy一次对象，代用一次对象长度的调用，后续处理逻辑类似； 因为 String 牵扯到编码方式，处理的时候略有不同。

2. Map的处理则有点像真正的遍历操作，使用 `mapiterinit(type, range, &hiter)` 初始化, mapiterinit

   的定义在 [hash_map.go](https://github.com/golang/gofrontend/blob/cef3934fbc63f5e121abb8f88d3799961ac95b59/libgo/go/runtime/hashmap.go#L727:4)，使用 `mapiternext(&hiter)` 来进行下一个遍历，[mapiternext的定义](https://github.com/golang/gofrontend/blob/cef3934fbc63f5e121abb8f88d3799961ac95b59/libgo/go/runtime/hashmap.go#L794)，所以行为上与其他两种类型有所不同，另外 go 1.9 以前，同时对 Map 进行读写操作，会报错 race 的错误 `concurrent map iteration and map write`，因此一般都是lock 或者 在同一个 goroutine 中对 maps 进行操作:

   * 在 `range` 中循环对 maps 做添加或者删除元素的操作是安全的；`maps` 实际上是结构体的指针。循环开始前，只会复制指针而不是内部的数据结构，因此在循环中添加或删除元素所操作的内存空间和原变量一致，合情合理。
   * 如果在 `range` 循环中对于 maps 添加了一个元素，那么这个元素未必一定出现在后续的迭代中；这是因为 map 底层结构为 hash table，后续添加的元素可能被添加到了已经遍历过的slot中；

3. Channel的处理则相对比较简单，就是一直从 channel 中读取数据，并检测 channel 是否已经被关闭；

## 4. Range 使用注意事项

因为 Range 的操作中涉及到对象的 Copy 操作，因为在遍历 String 和 Array 的时候如果不是采用指针的话，会造成 Copy 的性能。



通过 range 内部原理的分析，Dave's tweet的问题则可以很清晰来说明了，由于在 range 初始化的时候已经进行了对象 Copy 操作，程序显而易见会退出。

```go
for_temp := v
len_temp := len(for_temp)
for index_temp = 0; index_temp < len_temp; index_temp++ {
        value_temp = for_temp[index_temp]
        index = index_temp
        value = value_temp
        v = append(v, index)
}
```





## 参考：

1. [Iterating Over Slices In Go](https://www.ardanlabs.com/blog/2013/09/iterating-over-slices-in-go.html)
2. [gofrontend](https://github.com/golang/gofrontend)
3. [Go Range Loop Internals](https://garbagecollected.org/2017/02/22/go-range-loop-internals/)