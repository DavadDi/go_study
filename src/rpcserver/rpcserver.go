package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type Echo int

func (t *Echo) Hi(args string, reply *string) error {
	*reply = "echo:" + args
	return nil
}

type Args struct {
	A, B int
}

type Sum int

func (t *Sum) Sum(args *Args, reply *int) error {
	*reply = args.A + args.B
	return nil
}

func main() {
	rpc.Register(new(Echo))
	rpc.Register(new(Sum))

	rpc.HandleHTTP()

	l, e := net.Listen("tcp", ":1234")

	if e != nil {
		log.Fatal("listen error:", e)
	}

	http.Serve(l, nil)
}
