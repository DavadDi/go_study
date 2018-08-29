/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"log"
	"os"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"io/ioutil"
	"crypto/x509"
	"google.golang.org/grpc/credentials"
)

const (
	address     = "frontend.local:4443"
	defaultName = "world"
)

func main() {
	// Read cert file
	FrontendCert, _ := ioutil.ReadFile("../certs/frontend.crt")

	// Create CertPool
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(FrontendCert)

	// Create credentials
	credsClient := credentials.NewClientTLSFromCert(roots, "")

	// Set up a connection to the server.
	log.Printf("Ready to connect to %s\n", address)
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credsClient))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	/*
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credsClient))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	*/

	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}
