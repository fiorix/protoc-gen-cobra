// Copyright 2016 The protoc-gen-cobra authors. All rights reserved.
//
// Based on protoc-gen-go from https://github.com/golang/protobuf.
// Copyright 2015 The Go Authors.  All rights reserved.

/*
	protoc-gen-cobra is a plugin for the Google protocol buffer compiler to
	generate Go code to be used for building command line clients using cobra.
	Run it by building this program and putting it in your path with the name
		protoc-gen-cobra
	That word 'cobra' at the end becomes part of the option string set for the
	protocol compiler, so once the protocol compiler (protoc) is installed
	you can run
		protoc --cobra_out=output_directory input_directory/file.proto
	to generate Go bindings for the protocol defined by file.proto.
	With that input, the output will be written to
		output_directory/file.cobra.pb.go
	Use it combined with the grpc output
		protoc --go_out=plugins=grpc:. --cobra_out=plugins=client:. *.proto

	The generated code is documented in the package comment for
	the library.

	See the README and documentation for protocol buffers to learn more:
		https://developers.google.com/protocol-buffers/

*/
package documentation
