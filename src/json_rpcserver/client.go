package main

import (
	"fmt"
	"log"
	"net/rpc/jsonrpc"
)

type Args struct {
	A, B int
}

func main() {
	client, err := jsonrpc.Dial("tcp", "127.0.0.1:1234")

	if err != nil {
		log.Fatal("dialing:", err)
	}

	var str = "hello rpc"
	var reply string
	err = client.Call("Echo.Hi", str, &reply)

	if err != nil {
		log.Fatal("arith error:", err)
	}

	fmt.Printf("Echo: %s\n", reply)

	var args = &Args{7, 8}
	var sum int
	err = client.Call("Sum.Sum", args, &sum)

	if err != nil {
		log.Fatal("sum error:", err)
	}

	fmt.Printf("Sum: %d + %d = %d\n", args.A, args.B, sum)
}
