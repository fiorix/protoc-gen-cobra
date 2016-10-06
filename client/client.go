// Copyright 2016 The protoc-gen-cobra authors. All rights reserved.
//
// Based on protoc-gen-go from https://github.com/golang/protobuf.
// Copyright 2015 The Go Authors.  All rights reserved.

// Package client outputs a gRPC service client in Go code, using cobra.
// It runs as a plugin for the Go protocol buffer compiler plugin.
// It is linked in to protoc-gen-cobra.
package client

import (
	"bytes"
	"html/template"
	"path"
	"strconv"
	"strings"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/fiorix/protoc-gen-cobra/generator"
)

// generatedCodeVersion indicates a version of the generated code.
// It is incremented whenever an incompatibility between the generated code and
// the grpc package is introduced; the generated code references
// a constant, grpc.SupportPackageIsVersionN (where N is generatedCodeVersion).
const generatedCodeVersion = 3

func init() {
	generator.RegisterPlugin(new(client))
}

// client is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type client struct {
	gen *generator.Generator
}

// Name returns the name of this plugin, "client".
func (c *client) Name() string {
	return "client"
}

// map of import pkg name to unique name
type importPkg map[string]*pkgInfo

type pkgInfo struct {
	ImportPath string
	KnownType  string
	UniqueName string
}

var importPkgs = importPkg{
	"cobra":       {ImportPath: "github.com/spf13/cobra", KnownType: "Command"},
	"context":     {ImportPath: "context", KnownType: "Context"},
	"credentials": {ImportPath: "google.golang.org/grpc/credentials", KnownType: "AuthInfo"},
	"filepath":    {ImportPath: "path/filepath", KnownType: "WalkFunc"},
	"grpc":        {ImportPath: "google.golang.org/grpc", KnownType: "ClientConn"},
	"iocodec":     {ImportPath: "github.com/fiorix/protoc-gen-cobra/iocodec", KnownType: "Encoder"},
	"ioutil":      {ImportPath: "io/ioutil", KnownType: "=Discard"},
	"json":        {ImportPath: "encoding/json", KnownType: "Encoder"},
	"log":         {ImportPath: "log", KnownType: "Logger"},
	"os":          {ImportPath: "os", KnownType: "File"},
	"pflag":       {ImportPath: "github.com/spf13/pflag", KnownType: "FlagSet"},
	"template":    {ImportPath: "text/template", KnownType: "Template"},
	"time":        {ImportPath: "time", KnownType: "Time"},
	"tls":         {ImportPath: "crypto/tls", KnownType: "Config"},
	"x509":        {ImportPath: "crypto/x509", KnownType: "Certificate"},
}

// Init initializes the plugin.
func (c *client) Init(gen *generator.Generator) {
	c.gen = gen
	for k := range importPkgs {
		importPkgs[k].UniqueName = generator.RegisterUniquePackageName(k, nil)
	}
}

// P forwards to c.gen.P.
func (c *client) P(args ...interface{}) { c.gen.P(args...) }

// Generate generates code for the services in the given file.
func (c *client) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	c.P("// Reference imports to suppress errors if they are not otherwise used.")
	for _, v := range importPkgs {
		if strings.HasPrefix(v.KnownType, "=") {
			c.P("var _ = ", v.UniqueName, ".", v.KnownType[1:])
		} else {
			c.P("var _ ", v.UniqueName, ".", v.KnownType)
		}
	}

	// Assert version compatibility.
	c.P("// This is a compile-time assertion to ensure that this generated file")
	c.P("// is compatible with the grpc package it is being compiled against.")
	c.P("const _ = ", importPkgs["grpc"].UniqueName, ".SupportPackageIsVersion", generatedCodeVersion)
	c.P()

	for i, service := range file.FileDescriptorProto.Service {
		c.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (c *client) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	c.P("import (")
	for _, v := range importPkgs {
		c.P(v.UniqueName, " ", strconv.Quote(path.Join(c.gen.ImportPrefix, v.ImportPath)))
	}
	c.P(")")
	c.P()
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in gRPC?
}

// generateService generates all the code for the named service.
func (c *client) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	origServName := service.GetName()
	fullServName := origServName
	if pkg := file.GetPackage(); pkg != "" {
		fullServName = pkg + "." + fullServName
	}
	servName := generator.CamelCase(origServName)

	c.P()
	c.generateCommand(servName)
	c.P()
	for _, method := range service.Method {
		c.generateSubcommand(servName, method)
	}
	c.P()
}

var generateCommandTemplateCode = `
var _Default{{.Name}}ClientCommandConfig = _New{{.Name}}ClientCommandConfig()

func init() {
	_Default{{.Name}}ClientCommandConfig.AddFlags({{.Name}}ClientCommand.PersistentFlags())
}

type _{{.Name}}ClientCommandConfig struct {
	ServerAddr string
	RequestFile string
	PrintSampleRequest bool
	ResponseFormat string
	Timeout time.Duration
	TLS bool
	InsecureSkipVerify bool
	CACertFile string
	CertFile string
	KeyFile string
}

func _New{{.Name}}ClientCommandConfig() *_{{.Name}}ClientCommandConfig {
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}
	timeout, err := time.ParseDuration(os.Getenv("TIMEOUT"))
	if err != nil {
		timeout = 10 * time.Second
	}
	outfmt := os.Getenv("RESPONSE_FORMAT")
	if outfmt == "" {
		outfmt = "yaml"
	}
	return &_{{.Name}}ClientCommandConfig{
		ServerAddr: addr,
		RequestFile: os.Getenv("REQUEST_FILE"),
		PrintSampleRequest: os.Getenv("PRINT_SAMPLE_REQUEST") != "",
		ResponseFormat: outfmt,
		Timeout: timeout,
		TLS: os.Getenv("TLS") != "",
		InsecureSkipVerify: os.Getenv("TLS_INSECURE_SKIP_VERIFY") != "",
		CACertFile: os.Getenv("TLS_CA_CERT_FILE"),
		CertFile: os.Getenv("TLS_CERT_FILE"),
		KeyFile: os.Getenv("TLS_KEY_FILE"),
	}
}

func (o *_{{.Name}}ClientCommandConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.ServerAddr, "server-addr", "s", o.ServerAddr, "server address in form of host:port")
	fs.StringVarP(&o.RequestFile, "request-file", "f", o.RequestFile, "client request file (extension must be yaml, json, or xml)")
	fs.BoolVar(&o.PrintSampleRequest, "print-sample-request", o.PrintSampleRequest, "print sample request file and exit")
	fs.StringVarP(&o.ResponseFormat, "response-format", "o", o.ResponseFormat, "response format (yaml, json, or xml)")
	fs.DurationVar(&o.Timeout, "timeout", o.Timeout, "client connection timeout")
	fs.BoolVar(&o.TLS, "tls", o.TLS, "enable tls")
	fs.BoolVar(&o.InsecureSkipVerify, "tls-insecure-skip-verify", o.InsecureSkipVerify, "INSECURE: skip tls checks")
	fs.StringVar(&o.CACertFile, "tls-ca-cert-file", o.CACertFile, "ca certificate file")
	fs.StringVar(&o.CertFile, "tls-cert-file", o.CertFile, "client certificate file")
	fs.StringVar(&o.KeyFile, "tls-key-file", o.KeyFile, "client key file")
}

var {{.Name}}ClientCommand = &cobra.Command{
	Use: "{{.UseName}}",
}

func _Dial{{.Name}}() (*grpc.ClientConn, {{.Name}}Client, error) {
	cfg := _Default{{.Name}}ClientCommandConfig
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTimeout(cfg.Timeout),
	}
	if cfg.TLS {
		tlsConfig := tls.Config{}
		if cfg.InsecureSkipVerify {
			tlsConfig.InsecureSkipVerify = true
		}
		if cfg.CACertFile != "" {
			cacert, err := ioutil.ReadFile(cfg.CACertFile)
			if err != nil {
				return nil, nil, fmt.Errorf("ca cert: %v", err)
			}
			certpool := x509.NewCertPool()
			certpool.AppendCertsFromPEM(cacert)
			tlsConfig.RootCAs = certpool
		}
		if cfg.CertFile != "" {
			if cfg.KeyFile == "" {
				return nil, nil, fmt.Errorf("missing key file")
			}
			pair, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				return nil, nil, fmt.Errorf("cert/key: %v", err)
			}
			tlsConfig.Certificates = []tls.Certificate{pair}
		}
		cred := credentials.NewTLS(&tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(cred))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	conn, err := grpc.Dial(cfg.ServerAddr, opts...)
	if err != nil {
		return nil, nil, err
	}
	return conn, New{{.Name}}Client(conn), nil
}

type _{{.Name}}RoundTripFunc func(cli {{.Name}}Client) (out interface{}, err error)

func _{{.Name}}RoundTrip(v interface{}, fn _{{.Name}}RoundTripFunc) error {
		cfg := _Default{{.Name}}ClientCommandConfig
		var e iocodec.EncoderMaker
		var ok bool
		if cfg.ResponseFormat == "" {
			e = iocodec.DefaultEncoders["yaml"]
		} else {
			e, ok = iocodec.DefaultEncoders[cfg.ResponseFormat]
			if !ok {
				return fmt.Errorf("invalid response format: %q", cfg.ResponseFormat)
			}
		}
		if cfg.PrintSampleRequest {
			return e.NewEncoder(os.Stdout).Encode(v)
		}
		if cfg.RequestFile == "" {
			return fmt.Errorf("no request file")
		}
		ext := filepath.Ext(cfg.RequestFile)
		if len(ext) > 0 && ext[0] == '.' {
			ext = ext[1:]
		}
		d, ok := iocodec.DefaultDecoders[ext]
		if !ok {
			return fmt.Errorf("invalid request file format: %q", ext)
		}
		f, err := os.Open(cfg.RequestFile)
		if err != nil {
			return fmt.Errorf("request file: %v", err)
		}
		defer f.Close()
		err = d.NewDecoder(f).Decode(v)
		if err != nil {
			return fmt.Errorf("request parser: %v", err)
		}
		conn, client, err := _Dial{{.Name}}()
		if err != nil {
			return err
		}
		defer conn.Close()
		out, err := fn(client)
		if err != nil {
			return err
		}
		return e.NewEncoder(os.Stdout).Encode(out)
}
`

var generateCommandTemplate = template.Must(template.New("cmd").Parse(generateCommandTemplateCode))

func (c *client) generateCommand(servName string) {
	var b bytes.Buffer
	err := generateCommandTemplate.Execute(&b, struct {
		Name    string
		UseName string
	}{
		Name:    servName,
		UseName: strings.ToLower(servName),
	})
	if err != nil {
		c.gen.Error(err, "exec cmd template")
	}
	c.P(b.String())
	c.P()
}

var generateSubcommandTemplateCode = `
var _{{.FullName}}ClientCommand = &cobra.Command{
	Use: "{{.UseName}}",
	Run: func(cmd *cobra.Command, args []string) {
		var in {{.InputType}}
		err := _{{.ServiceName}}RoundTrip(&in, func(cli {{.ServiceName}}Client) (interface{}, error) {
			return cli.{{.Name}}(context.Background(), &in)
		})
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	{{.ServiceName}}ClientCommand.AddCommand(_{{.FullName}}ClientCommand)
}
`

var generateSubcommandTemplate = template.Must(template.New("subcmd").Parse(generateSubcommandTemplateCode))

func (c *client) generateSubcommand(servName string, method *pb.MethodDescriptorProto) {
	if method.GetClientStreaming() || method.GetServerStreaming() {
		return // TODO: handle streams correctly
	}
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}
	_, typ := path.Split(method.GetInputType())
	typ = strings.SplitN(typ, ".", 3)[2] // TODO: get type name without pkg
	var b bytes.Buffer
	err := generateSubcommandTemplate.Execute(&b, struct {
		Name        string
		UseName     string
		ServiceName string
		FullName    string
		InputType   string
	}{
		Name:        methName,
		UseName:     strings.ToLower(methName),
		ServiceName: servName,
		FullName:    servName + methName,
		InputType:   typ,
	})
	if err != nil {
		c.gen.Error(err, "exec subcmd template")
	}
	c.P(b.String())
	c.P()
}
