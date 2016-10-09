package main

import (
	"io"
	"sync"

	"golang.org/x/net/context"

	"github.com/fiorix/protoc-gen-cobra/example/pb"
)

type Cache struct {
	mu sync.Mutex
	kv map[string]string
}

func NewCache() *Cache {
	return &Cache{
		kv: make(map[string]string),
	}
}

func (c *Cache) Set(ctx context.Context, in *pb.SetRequest) (*pb.SetResponse, error) {
	c.mu.Lock()
	c.kv[in.Key] = in.Value
	c.mu.Unlock()
	return &pb.SetResponse{}, nil
}

func (c *Cache) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return &pb.GetResponse{
		Value: c.kv[in.Key],
	}, nil
}

func (c *Cache) MultiSet(stream pb.Cache_MultiSetServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		c.mu.Lock()
		c.kv[req.Key] = req.Value
		c.mu.Unlock()
	}
	return stream.SendAndClose(&pb.SetResponse{})
}

func (c *Cache) MultiGet(stream pb.Cache_MultiGetServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		c.mu.Lock()
		v := c.kv[req.Key]
		c.mu.Unlock()
		stream.Send(&pb.GetResponse{
			Value: v,
		})
	}
}
