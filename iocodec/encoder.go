package iocodec

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"

	"gopkg.in/yaml.v2"
)

// DefaultEncoders contains the default list of encoders per MIME type.
var DefaultEncoders = EncoderGroup{
	"xml":        EncoderMakerFunc(func(w io.Writer) Encoder { return &xmlEncoder{w} }),
	"json":       EncoderMakerFunc(func(w io.Writer) Encoder { return &jsonEncoder{w, false} }),
	"prettyjson": EncoderMakerFunc(func(w io.Writer) Encoder { return &jsonEncoder{w, true} }),
	"yaml":       EncoderMakerFunc(func(w io.Writer) Encoder { return &yamlEncoder{w} }),
}

type (
	// An Encoder encodes data from v.
	Encoder interface {
		Encode(v interface{}) error
	}

	// An EncoderGroup maps MIME types to EncoderMakers.
	EncoderGroup map[string]EncoderMaker

	// An EncoderMaker creates and returns a new Encoder.
	EncoderMaker interface {
		NewEncoder(w io.Writer) Encoder
	}

	// EncoderMakerFunc is an adapter for creating EncoderMakers
	// from functions.
	EncoderMakerFunc func(w io.Writer) Encoder
)

// NewEncoder implements the EncoderMaker interface.
func (f EncoderMakerFunc) NewEncoder(w io.Writer) Encoder {
	return f(w)
}

type xmlEncoder struct {
	w io.Writer
}

func (xe *xmlEncoder) Encode(v interface{}) error {
	xe.w.Write([]byte(xml.Header))
	defer xe.w.Write([]byte("\n"))
	e := xml.NewEncoder(xe.w)
	e.Indent("", "\t")
	return e.Encode(v)
}

type jsonEncoder struct {
	w      io.Writer
	pretty bool
}

func (je *jsonEncoder) Encode(v interface{}) error {
	if je.pretty {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		var out bytes.Buffer
		err = json.Indent(&out, b, "", "\t")
		if err != nil {
			return err
		}
		_, err = io.Copy(je.w, &out)
		if err != nil {
			return err
		}
		_, err = je.w.Write([]byte("\n"))
		return err
	}
	return json.NewEncoder(je.w).Encode(v)
}

type yamlEncoder struct {
	w io.Writer
}

func (ye *yamlEncoder) Encode(v interface{}) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = ye.w.Write(b)
	return err
}
