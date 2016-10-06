package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/fiorix/protoc-gen-cobra/example/pb"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	opts := []grpc.ServerOption{}
	srv := grpc.NewServer(opts...)
	pb.RegisterBankServer(srv, NewBank())
	pb.RegisterCacheServer(srv, NewCache())
	pb.RegisterTimerServer(srv, NewTimer())
	err = srv.Serve(ln)
	if err != nil {
		log.Fatal(err)
	}
}
