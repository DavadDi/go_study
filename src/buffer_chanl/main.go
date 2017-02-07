package main

import (
    "fmt"
    "time"
)

const MAX_SIZE = 3

func main() {
    resp := make(chan string, MAX_SIZE)

    for i := 1; i <= MAX_SIZE; i++ {
        go func(seq int) {
            secs := time.Duration(seq) * time.Second
            time.Sleep(secs)
            resp <- fmt.Sprintf("[%s] %d done!", time.Now().String(), seq)

        }(i) // notice, i must passed
    }

    start := time.Now()

    for i := 0; i < MAX_SIZE; i++ {
        fmt.Println(<-resp)

    }

    fmt.Println(time.Since(start).String())
}

