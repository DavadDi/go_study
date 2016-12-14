package main

import (
    "fmt"
    "runtime"
)

func main() {
    hello()
}

func hello() {
    pc, file, line, ok := runtime.Caller(0);
    if !ok {
        return
    }

    // get Fun object from pc
    fun := runtime.FuncForPC(pc)
    funcNmae := fun.Name()

    fmt.Printf("[%s:%d] Func name %s\n", file, line, funcNmae)

    pc, file, line, ok = runtime.Caller(1)
    if !ok {
        return
    }

    // get Fun object from pc
    fun = runtime.FuncForPC(pc)
    funcNmae = fun.Name()

    fmt.Printf("[%s:%d] Func name %s\n", file, line, funcNmae)

}

