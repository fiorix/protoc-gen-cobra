# protoc-gen-cobra

Cobra command line tool generator for Go gRPC.

[![GoDoc](https://godoc.org/github.com/fiorix/protoc-gen-cobra?status.svg)](https://godoc.org/github.com/fiorix/protoc-gen-cobra)

### What's this?

A plugin for the [protobuf](https://github.com/google/protobuf) compiler protoc, that generates Go code using [cobra](https://github.com/spf13/cobra). It is capable of generating client code for command line tools consistent with your protobuf description.

This:

```
service Bank {
	rpc Deposit(DepositRequest) returns (DepositReply)
}
```

produces a client like:

```
command bank deposit -f request.yaml -o json
```

It generates one [cobra.Command](https://godoc.org/github.com/spf13/cobra#Command) per gRPC service (e.g. bank), for you to import in your code. The service's rpc methods are sub-commands, and share the same command line semantics. They take a request file for input, and print the response in the specified format. The client currently supports basic connectivity settings such as tls on/off, tls client authentication and so on.

```
$ ./example bank
Usage:
  example bank [command]

Available Commands:
  deposit

Flags:
  -h, --help                       help for bank
      --print-sample-request       print sample request file and exit
  -f, --request-file string        client request file (extension must be yaml, json, or xml)
  -o, --response-format string     response format (yaml, json, or xml) (default "yaml")
  -s, --server-addr string         server address in form of host:port (default "localhost:8080")
      --timeout duration           client connection timeout (default 10s)
      --tls                        enable tls
      --tls-ca-cert-file string    ca certificate file
      --tls-cert-file string       client certificate file
      --tls-insecure-skip-verify   INSECURE: skip tls checks
      --tls-key-file string        client key file
```

This is an experiment. Was bored of writing the same boilerplate code to interact with gRPC servers, wanted something like [kubectl](http://kubernetes.io/docs/user-guide/kubectl-overview/). At some point I might want to generate server code too, similar to what go-swagger does. Perhaps look at using go-openapi too. Tests are lacking.
