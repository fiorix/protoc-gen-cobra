package main

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/fiorix/protoc-gen-cobra/example/pb"
)

type Timer struct{}

func NewTimer() *Timer { return &Timer{} }

func (t *Timer) Tick(in *pb.TickRequest, stream pb.Timer_TickServer) error {
	if in.Interval < 1 {
		return grpc.Errorf(codes.InvalidArgument, "interval param must be greater than 0")
	}
	now := time.Now()
	interval := time.Duration(in.Interval) * time.Second
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for range tick.C {
		now = now.Add(interval)
		err := stream.Send(&pb.TickResponse{
			Time: now.String(),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
