package main

import (
	"github.com/fiorix/protoc-gen-cobra/example/cmd"
	"github.com/fiorix/protoc-gen-cobra/example/pb"
)

func init() {
	// Add client generated commands to cobra's root cmd.
	cmd.RootCmd.AddCommand(pb.AuthClientCommand)
	cmd.RootCmd.AddCommand(pb.StoreClientCommand)
}
