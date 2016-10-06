package main

import (
	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/fiorix/protoc-gen-cobra/example/pb"
)

type Bank struct {
	mu             sync.Mutex
	accountBalance map[string]float64
}

func NewBank() *Bank {
	return &Bank{
		accountBalance: make(map[string]float64),
	}
}

func (b *Bank) Deposit(ctx context.Context, in *pb.DepositRequest) (*pb.DepositReply, error) {
	if in.Account == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "missing account name")
	}
	b.mu.Lock()
	v := b.accountBalance[in.Account] + in.Amount
	b.accountBalance[in.Account] = v
	b.mu.Unlock()
	reply := &pb.DepositReply{
		Account: in.Account,
		Balance: v,
	}
	return reply, nil
}
