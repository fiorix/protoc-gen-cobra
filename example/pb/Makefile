all:
	protoc \
		-I. \
		--gofast_out=plugins=grpc:. \
		--cobra_out=plugins=client:. \
		*.proto

clean:
	rm -f *.pb.go

deps:
	go get github.com/gogo/protobuf/protoc-gen-gofast
	go get github.com/fiorix/protoc-gen-cobra
