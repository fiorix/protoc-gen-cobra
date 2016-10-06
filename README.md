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

message DepositRequest {
	string account = 1;
	double amount = 2;
}

message DepositReply {
	string account = 1;
	double balance = 2;
}
```

produces a client that can do:

```
echo '{"account":"foobar","amount":10}' | command bank deposit
```

It generates one [cobra.Command](https://godoc.org/github.com/spf13/cobra#Command) per gRPC service (e.g. bank). The service's rpc methods are sub-commands, and share the same command line semantics. They take a request file for input, or stdin, and prints the response to the terminal, in the specified format. The client currently supports basic connectivity settings such as tls on/off, tls client authentication and so on.

```
$ ./example bank
Usage:
  example bank [command]

Available Commands:
  deposit

Flags:
  -h, --help                       help for bank
  -p, --print-sample-request       print sample request file and exit
  -f, --request-file string        client request file (must be json, yaml, or xml); use "-" for stdin + json
  -o, --response-format string     response format (json, prettyjson, yaml, or xml) (default "json")
  -s, --server-addr string         server address in form of host:port (default "localhost:8080")
      --timeout duration           client connection timeout (default 10s)
      --tls                        enable tls
      --tls-ca-cert-file string    ca certificate file
      --tls-cert-file string       client certificate file
      --tls-insecure-skip-verify   INSECURE: skip tls checks
      --tls-key-file string        client key file
```

This is an experiment. Was bored of writing the same boilerplate code to interact with gRPC servers, wanted something like [kubectl](http://kubernetes.io/docs/user-guide/kubectl-overview/). At some point I might want to generate server code too, similar to what go-swagger does. Perhaps look at using go-openapi too. Tests are lacking.

### Streams

gRPC client and server streams are supported. On server streams, each response is printed out using the specified response format. Client streams must be formatted as json, one document per line, from a file or stdin.

Example:

```
$ cat req.json
{"key":"hello","value":"world"}
{"key":"foo","value":"bar"}

$ ./example cache multiset -f req.json

...

$ echo '{"key":"hello"}' | ./example cache get
{"value":"world"}
```
