# 使用 go build 修改go代码中的变量



## 1. 需求场景

编译 go 相关代码的时候，可能需要在编译的版本中添加 编译的 git commit sha1、编译时间等各种随着编译变化的信息。



```
package version

var (  
    Version   = "Not Provide"
    BuildTime = "Not Provide"
    GitSHA    = "Not Provide"
)
```



在编译的时候需要根据编译的 git sha1或者 version 实时进行替换



## 2. 实现方式

go build 中 可以通过 ldconfig 在编译过程中实时修改 pkg version 变量值，来达到上述目的：



main.go

```go
package main

import "fmt"
import "my_version/version"

var Version = "not provider"

func  main() {
    fmt.Println(Version)
    fmt.Println(version.GitSHA)
}
```



version/version.go

```go
package version

var GitSHA="Not provide"
```



build.sh

```shell

#!/bin/bash
set -x

export GOPATH="`pwd`/../.."

GIT_SHA=`git rev-parse --short HEAD`
VERSION="0.1"

go build -ldflags " -X main.Version=${VERSION} -X my_version/version.GitSHA=${GIT_SHA}" main.go
```