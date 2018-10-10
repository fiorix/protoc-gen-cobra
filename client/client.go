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
	"fmt"
	"html/template"
	"path"
	"sort"
	"strconv"
	"strings"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/fiorix/protoc-gen-cobra/generator"
)

// generatedCodeVersion indicates a version of the generated code.
// It is incremented whenever an incompatibility between the generated code and
// the grpc package is introduced; the generated code references
// a constant, grpc.SupportPackageIsVersionN (where N is generatedCodeVersion).
const generatedCodeVersion = 4

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

var importPkgsByName = importPkg{
	"cobra":       {ImportPath: "github.com/spf13/cobra", KnownType: "Command"},
	"context":     {ImportPath: "golang.org/x/net/context", KnownType: "Context"},
	"credentials": {ImportPath: "google.golang.org/grpc/credentials", KnownType: "AuthInfo"},
	"envconfig":   {ImportPath: "github.com/kelseyhightower/envconfig", KnownType: "Decoder"},
	"filepath":    {ImportPath: "path/filepath", KnownType: "WalkFunc"},
	"grpc":        {ImportPath: "google.golang.org/grpc", KnownType: "ClientConn"},
	"io":          {ImportPath: "io", KnownType: "Reader"},
	"iocodec":     {ImportPath: "github.com/fiorix/protoc-gen-cobra/iocodec", KnownType: "Encoder"},
	"ioutil":      {ImportPath: "io/ioutil", KnownType: "=Discard"},
	"json":        {ImportPath: "encoding/json", KnownType: "Encoder"},
	"log":         {ImportPath: "log", KnownType: "Logger"},
	"net":         {ImportPath: "net", KnownType: "IP"},
	"oauth":       {ImportPath: "google.golang.org/grpc/credentials/oauth", KnownType: "TokenSource"},
	"oauth2":      {ImportPath: "golang.org/x/oauth2", KnownType: "Token"},
	"os":          {ImportPath: "os", KnownType: "File"},
	"pflag":       {ImportPath: "github.com/spf13/pflag", KnownType: "FlagSet"},
	"template":    {ImportPath: "text/template", KnownType: "Template"},
	"time":        {ImportPath: "time", KnownType: "Time"},
	"tls":         {ImportPath: "crypto/tls", KnownType: "Config"},
	"x509":        {ImportPath: "crypto/x509", KnownType: "Certificate"},
}
var sortedImportPkgNames = make([]string, 0, len(importPkgsByName))

// Init initializes the plugin.
func (c *client) Init(gen *generator.Generator) {
	c.gen = gen
	for k := range importPkgsByName {
		importPkgsByName[k].UniqueName = generator.RegisterUniquePackageName(k, nil)
		sortedImportPkgNames = append(sortedImportPkgNames, k)
	}
	sort.Strings(sortedImportPkgNames)
}

// P forwards to c.gen.P.
func (c *client) P(args ...interface{}) { c.gen.P(args...) }

// Generate generates code for the services in the given file.
func (c *client) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	c.P("// Reference imports to suppress errors if they are not otherwise used.")
	for _, n := range sortedImportPkgNames {
		v := importPkgsByName[n]
		if strings.HasPrefix(v.KnownType, "=") {
			c.P("var _ = ", v.UniqueName, ".", v.KnownType[1:])
		} else {
			c.P("var _ ", v.UniqueName, ".", v.KnownType)
		}
	}

	// Assert version compatibility.
	c.P("// This is a compile-time assertion to ensure that this generated file")
	c.P("// is compatible with the grpc package it is being compiled against.")
	c.P("const _ = ", importPkgsByName["grpc"].UniqueName, ".SupportPackageIsVersion", generatedCodeVersion)
	c.P()

	for i, service := range file.FileDescriptorProto.Service {
		c.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (c *client) GenerateImports(file *generator.FileDescriptor, imports []*generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	c.P("import (")
	for _, n := range sortedImportPkgNames {
		v := importPkgsByName[n]
		c.P(v.UniqueName, " ", strconv.Quote(path.Join(c.gen.ImportPrefix, v.ImportPath)))
	}

	importPathByPackage := map[string]string{}
	for _, imp := range imports {
		if *file.Package == *imp.Package {
			continue
		}
		if imp.FileDescriptorProto.GetOptions().GetGoPackage() != "" {
			importPathByPackage[*imp.FileDescriptorProto.Package] = strconv.Quote(*imp.FileDescriptorProto.Options.GoPackage)
		} else {
			importPathByPackage[*imp.FileDescriptorProto.Package] = strconv.Quote(path.Join(c.gen.ImportPrefix, *imp.FileDescriptorProto.Package))
		}
	}

	importedPackagesByName := map[string]string{}
	for _, service := range file.FileDescriptorProto.Service {
		for _, method := range service.Method {
			importName, pkg, _ := inputNames(method.GetInputType())

			if importPath, found := importPathByPackage[pkg]; found {
				importedPackagesByName[importName] = importPath
			}
		}
	}
	importedPackageNames := make([]string, 0, len(importedPackagesByName))
	for n := range importedPackagesByName {
		importedPackageNames = append(importedPackageNames, n)
	}
	sort.Strings(importedPackageNames)
	for _, n := range importedPackageNames {
		c.P(n, " ", importedPackagesByName[n])
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
		c.generateSubcommand(servName, file, method)
	}
	c.P()
}

var generateCommandTemplateCode = `
var _Default{{.Name}}ClientCommandConfig = _New{{.Name}}ClientCommandConfig()

type _{{.Name}}ClientCommandConfig struct {
	ServerAddr string	` + "`" + `envconfig:"SERVER_ADDR" default:"localhost:8080"` + "`" + `
	RequestFile string	` + "`" + `envconfig:"REQUEST_FILE"` + "`" + `
	PrintSampleRequest bool	` + "`" + `envconfig:"PRINT_SAMPLE_REQUEST"` + "`" + `
	ResponseFormat string	` + "`" + `envconfig:"RESPONSE_FORMAT" default:"json"` + "`" + `
	Timeout time.Duration	` + "`" + `envconfig:"TIMEOUT" default:"10s"` + "`" + `
	TLS bool		` + "`" + `envconfig:"TLS"` + "`" + `
	ServerName string	` + "`" + `envconfig:"TLS_SERVER_NAME"` + "`" + `
	InsecureSkipVerify bool	` + "`" + `envconfig:"TLS_INSECURE_SKIP_VERIFY"` + "`" + `
	CACertFile string	` + "`" + `envconfig:"TLS_CA_CERT_FILE"` + "`" + `
	CertFile string		` + "`" + `envconfig:"TLS_CERT_FILE"` + "`" + `
	KeyFile string		` + "`" + `envconfig:"TLS_KEY_FILE"` + "`" + `
	AuthToken string	` + "`" + `envconfig:"AUTH_TOKEN"` + "`" + `
	AuthTokenType string	` + "`" + `envconfig:"AUTH_TOKEN_TYPE" default:"Bearer"` + "`" + `
	JWTKey string		` + "`" + `envconfig:"JWT_KEY"` + "`" + `
	JWTKeyFile string	` + "`" + `envconfig:"JWT_KEY_FILE"` + "`" + `
}

func _New{{.Name}}ClientCommandConfig() *_{{.Name}}ClientCommandConfig {
	c := &_{{.Name}}ClientCommandConfig{}
	envconfig.Process("", c)
	return c
}

func (o *_{{.Name}}ClientCommandConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.ServerAddr, "server-addr", "s", o.ServerAddr, "server address in form of host:port")
	fs.StringVarP(&o.RequestFile, "request-file", "f", o.RequestFile, "client request file (must be json, yaml, or xml); use \"-\" for stdin + json")
	fs.BoolVarP(&o.PrintSampleRequest, "print-sample-request", "p", o.PrintSampleRequest, "print sample request file and exit")
	fs.StringVarP(&o.ResponseFormat, "response-format", "o", o.ResponseFormat, "response format (json, prettyjson, yaml, or xml)")
	fs.DurationVar(&o.Timeout, "timeout", o.Timeout, "client connection timeout")
	fs.BoolVar(&o.TLS, "tls", o.TLS, "enable tls")
	fs.StringVar(&o.ServerName, "tls-server-name", o.ServerName, "tls server name override")
	fs.BoolVar(&o.InsecureSkipVerify, "tls-insecure-skip-verify", o.InsecureSkipVerify, "INSECURE: skip tls checks")
	fs.StringVar(&o.CACertFile, "tls-ca-cert-file", o.CACertFile, "ca certificate file")
	fs.StringVar(&o.CertFile, "tls-cert-file", o.CertFile, "client certificate file")
	fs.StringVar(&o.KeyFile, "tls-key-file", o.KeyFile, "client key file")
	fs.StringVar(&o.AuthToken, "auth-token", o.AuthToken, "authorization token")
	fs.StringVar(&o.AuthTokenType, "auth-token-type", o.AuthTokenType, "authorization token type")
	fs.StringVar(&o.JWTKey, "jwt-key", o.JWTKey, "jwt key")
	fs.StringVar(&o.JWTKeyFile, "jwt-key-file", o.JWTKeyFile, "jwt key file")
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
		tlsConfig := &tls.Config{}
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
		if cfg.ServerName != "" {
			tlsConfig.ServerName = cfg.ServerName
		} else {
			addr, _, _ := net.SplitHostPort(cfg.ServerAddr)
			tlsConfig.ServerName = addr
		}
		//tlsConfig.BuildNameToCertificate()
		cred := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(cred))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	if cfg.AuthToken != "" {
		cred := oauth.NewOauthAccess(&oauth2.Token{
			AccessToken: cfg.AuthToken,
			TokenType: cfg.AuthTokenType,
		})
		opts = append(opts, grpc.WithPerRPCCredentials(cred))
	}
	if cfg.JWTKey != "" {
		cred, err := oauth.NewJWTAccessFromKey([]byte(cfg.JWTKey))
		if err != nil {
			return nil, nil, fmt.Errorf("jwt key: %v", err)
		}
		opts = append(opts, grpc.WithPerRPCCredentials(cred))
	}
	if cfg.JWTKeyFile != "" {
		cred, err := oauth.NewJWTAccessFromFile(cfg.JWTKeyFile)
		if err != nil {
			return nil, nil, fmt.Errorf("jwt key file: %v", err)
		}
		opts = append(opts, grpc.WithPerRPCCredentials(cred))
	}
	conn, err := grpc.Dial(cfg.ServerAddr, opts...)
	if err != nil {
		return nil, nil, err
	}
	return conn, New{{.Name}}Client(conn), nil
}

type _{{.Name}}RoundTripFunc func(cli {{.Name}}Client, in iocodec.Decoder, out iocodec.Encoder) error

func _{{.Name}}RoundTrip(sample interface{}, fn _{{.Name}}RoundTripFunc) error {
	cfg := _Default{{.Name}}ClientCommandConfig
	var em iocodec.EncoderMaker
	var ok bool
	if cfg.ResponseFormat == "" {
		em = iocodec.DefaultEncoders["json"]
	} else {
		em, ok = iocodec.DefaultEncoders[cfg.ResponseFormat]
		if !ok {
			return fmt.Errorf("invalid response format: %q", cfg.ResponseFormat)
		}
	}
	if cfg.PrintSampleRequest {
		return em.NewEncoder(os.Stdout).Encode(sample)
	}
	var d iocodec.Decoder
	if cfg.RequestFile == "" || cfg.RequestFile == "-" {
		d = iocodec.DefaultDecoders["json"].NewDecoder(os.Stdin)
	} else {
		f, err := os.Open(cfg.RequestFile)
		if err != nil {
			return fmt.Errorf("request file: %v", err)
		}
		defer f.Close()
		ext := filepath.Ext(cfg.RequestFile)
		if len(ext) > 0 && ext[0] == '.' {
			ext = ext[1:]
		}
		dm, ok := iocodec.DefaultDecoders[ext]
		if !ok {
			return fmt.Errorf("invalid request file format: %q", ext)
		}
		d = dm.NewDecoder(f)
	}
	conn, client, err := _Dial{{.Name}}()
	if err != nil {
		return err
	}
	defer conn.Close()
	return fn(client, d, em.NewEncoder(os.Stdout))
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
	Long: "{{.Name}} client\n\nYou can use environment variables with the same name of the command flags.\nAll caps and s/-/_, e.g. SERVER_ADDR.",
	Example: ` + "`" + `
Save a sample request to a file (or refer to your protobuf descriptor to create one):
	{{.UseName}} -p > req.json

Submit request using file:
	{{.UseName}} -f req.json

Authenticate using the Authorization header (requires transport security):
	export AUTH_TOKEN=your_access_token
	export SERVER_ADDR=api.example.com:443
	echo '{json}' | {{.UseName}} --tls` + "`" + `,
	Run: func(cmd *cobra.Command, args []string) {
		var v {{ with .InputPackage }}{{ . }}.{{ end }}{{.InputType}}
		err := _{{.ServiceName}}RoundTrip(v, func(cli {{.ServiceName}}Client, in iocodec.Decoder, out iocodec.Encoder) error {
{{if .ClientStream}}
			stream, err := cli.{{.Name}}(context.Background())
			if err != nil {
				return err
			}
			for {
				err = in.Decode(&v)
				if err == io.EOF {
					stream.CloseSend()
					break
				}
				if err != nil {
					return err
				}
				err = stream.Send(&v)
				if err != nil {
					return err
				}
			}
{{else}}
			err := in.Decode(&v)
			if err != nil {
				return err
			}
			{{if .ServerStream}}
			stream, err := cli.{{.Name}}(context.Background(), &v)
			{{else}}
			resp, err := cli.{{.Name}}(context.Background(), &v)
			{{end}}
			if err != nil {
				return err
			}
{{end}}
{{if .ServerStream}}
			for {
				v, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				err = out.Encode(v)
				if err != nil {
					return err
				}
			}
			return nil
{{else}}
			{{if .ClientStream}}
			resp, err := stream.CloseAndRecv()
			if err != nil {
				return err
			}
			{{end}}
			return out.Encode(resp)
{{end}}
		})
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	{{.ServiceName}}ClientCommand.AddCommand(_{{.FullName}}ClientCommand)
	_Default{{.ServiceName}}ClientCommandConfig.AddFlags(_{{.FullName}}ClientCommand.Flags())
}
`

var generateSubcommandTemplate = template.Must(template.New("subcmd").Parse(generateSubcommandTemplateCode))

func (c *client) generateSubcommand(servName string, file *generator.FileDescriptor, method *pb.MethodDescriptorProto) {
	/*
		if method.GetClientStreaming() || method.GetServerStreaming() {
			return // TODO: handle streams correctly
		}
	*/
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}
	importName, inputPackage, inputType := inputNames(method.GetInputType())
	if inputPackage == file.PackageName() {
		importName = ""
	}
	var b bytes.Buffer
	err := generateSubcommandTemplate.Execute(&b, struct {
		Name         string
		UseName      string
		ServiceName  string
		FullName     string
		InputPackage string
		InputType    string
		ClientStream bool
		ServerStream bool
	}{
		Name:         methName,
		UseName:      strings.ToLower(methName),
		ServiceName:  servName,
		FullName:     servName + methName,
		InputPackage: importName,
		InputType:    inputType,
		ClientStream: method.GetClientStreaming(),
		ServerStream: method.GetServerStreaming(),
	})
	if err != nil {
		c.gen.Error(err, "exec subcmd template")
	}
	c.P(b.String())
	c.P()
}

func inputNames(s string) (importName, inputPackage, inputType string) {
	_, typ := path.Split(s) // e.g. `.pkg.Type`
	typz := strings.Split(strings.Trim(typ, `.`), ".")
	if len(typz) < 2 {
		return
	}
	typeIdx := len(typz) - 1

	// .pkg.subpkg.Type -> pkg_subpkg_pb
	importName = fmt.Sprintf("%s_pb", strings.Join(typz[:typeIdx], `_`))

	// .pkg.subpkg.Type -> pkg.subpkg
	inputPackage = strings.Join(typz[:typeIdx], `.`)

	// .pkg.subpkg.Type -> Type
	inputType = typz[typeIdx]

	return
}
