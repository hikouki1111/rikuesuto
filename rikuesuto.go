package rikuesuto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strings"
)

type Part struct {
	MIMEHeader textproto.MIMEHeader
	Body       io.Reader
}

// MultipartData Files is map of "file name": "file path".
//
// Parts is list of Part.
//
// Fields is map of "field name": io.Reader.
//
// You must fill Files / Parts / Fields.
//
// Optional Boundary.
//
// You don't need to fill ContentType and Buffer.
type MultipartData struct {
	Files       map[string]*os.File
	Parts       []Part
	Fields      map[string]io.Reader
	Boundary    string
	ContentType string
	Buffer      *bytes.Buffer
}

type Config struct {
	URL         string
	Header      http.Header
	JSON        map[string]interface{}
	Form        map[string]string
	Multipart   *MultipartData
	Text        string
	OctetStream *os.File
}

type ContentTypeEnum int

const (
	NULL ContentTypeEnum = iota
	JSON
	Form
	Multipart
	Text
	Octet
)

func (co ContentTypeEnum) GetContentType() string {
	switch co {
	case NULL:
		return ""
	case JSON:
		return "application/json"
	case Form:
		return "application/x-www-form-urlencoded"
	case Multipart:
		return "multipart/form-data"
	case Text:
		return "text/plain"
	case Octet:
		return "application/octet-stream"
	}
	return ""
}

func (c Config) GetData() (string, io.Reader) {
	contentType := NULL
	if c.JSON != nil {
		contentType = JSON
	}
	if c.Form != nil {
		if contentType != NULL {
			panic("config cannot contain more than 1 data")
		}
		contentType = Form
	}
	if c.Multipart != nil {
		if contentType != NULL {
			panic("config cannot contain more than 1 data")
		}
		contentType = Multipart

		buffer := &bytes.Buffer{}
		mw := multipart.NewWriter(buffer)
		if c.Multipart.Boundary != "" {
			err := mw.SetBoundary(c.Multipart.Boundary)
			if err != nil {
				panic(err)
			}
		}

		if c.Multipart.Files != nil {
			for filename, file := range c.Multipart.Files {
				fw, err := mw.CreateFormFile("file", filename)
				if err != nil {
					panic(err)
				}
				_, err = io.Copy(fw, file)
				if err != nil {
					panic(err)
				}
			}
		}

		if c.Multipart.Parts != nil {
			for _, part := range c.Multipart.Parts {
				pw, err := mw.CreatePart(part.MIMEHeader)
				if err != nil {
					panic(err)
				}
				_, err = io.Copy(pw, part.Body)
				if err != nil {
					panic(err)
				}
			}
		}

		if c.Multipart.Fields != nil {
			for fieldName, buf := range c.Multipart.Fields {
				fw, err := mw.CreateFormField(fieldName)
				if err != nil {
					panic(err)
				}
				_, err = io.Copy(fw, buf)
				if err != nil {
					panic(err)
				}
			}
		}

		c.Multipart.ContentType = mw.FormDataContentType()
		err := mw.Close()
		if err != nil {
			panic(err)
		}
		c.Multipart.Buffer = buffer
	}
	if c.Text != "" {
		if contentType != NULL {
			panic("config cannot contain more than 1 data")
		}
		contentType = Text
	}
	if c.OctetStream != nil {
		if contentType != NULL {
			panic("config cannot contain more than 1 data")
		}
		contentType = Octet
	}

	switch contentType {
	case NULL:
		return "", nil
	case JSON:
		data, err := json.Marshal(c.JSON)
		if err != nil {
			panic("invalid json")
		}
		return JSON.GetContentType(), bytes.NewBuffer(data)
	case Form:
		values := url.Values{}
		for k, v := range c.Form {
			values.Add(k, v)
		}
		return Form.GetContentType(), strings.NewReader(values.Encode())
	case Multipart:
		fmt.Println(*c.Multipart.Buffer)
		return c.Multipart.ContentType, c.Multipart.Buffer
	case Text:
		return Text.GetContentType(), strings.NewReader(c.Text)
	case Octet:
		return Octet.GetContentType(), c.OctetStream
	}
	return "", nil
}

func GetMIMEContentType(value string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", value)
	return header
}

func NewRequest(method string, config *Config) (*http.Request, error) {
	contentType, body := config.GetData()
	req, err := http.NewRequest(method, config.URL, body)
	if err == nil && contentType != "" {
		if config.Header != nil {
			req.Header = config.Header
		}
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", contentType)
		}
	}
	return req, err
}

func MustNewRequest(method string, config *Config) *http.Request {
	req, err := NewRequest(method, config)
	if err != nil {
		panic(err)
	}
	return req
}

func MustDo(client *http.Client, req *http.Request) *http.Response {
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	return res
}

func DoReadBody(client *http.Client, req *http.Request) ([]byte, *http.Response, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	b, err := ReadBody(res)
	if err != nil {
		return nil, nil, err
	}
	return b, res, nil
}

func MustDoReadBody(client *http.Client, req *http.Request) ([]byte, *http.Response) {
	b, res, err := DoReadBody(client, req)
	if err != nil {
		panic(err)
	}
	return b, res
}

func DoReadString(client *http.Client, req *http.Request) (string, *http.Response, error) {
	b, res, err := DoReadBody(client, req)
	if err != nil {
		return "", nil, err
	}
	return string(b), res, nil
}

func MustDoReadString(client *http.Client, req *http.Request) (string, *http.Response) {
	s, res, err := DoReadString(client, req)
	if err != nil {
		panic(err)
	}
	return s, res
}

func ReadBody(res *http.Response) ([]byte, error) {
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func MustReadBody(res *http.Response) []byte {
	b, err := ReadBody(res)
	if err != nil {
		panic(err)
	}
	return b
}

func ReadString(res *http.Response) (string, error) {
	b, err := ReadBody(res)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func MustReadString(res *http.Response) string {
	s, err := ReadString(res)
	if err != nil {
		panic(err)
	}
	return s
}

func Get(config *Config) (*http.Request, error) {
	req, err := NewRequest("GET", config)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MustGet(config *Config) *http.Request {
	req, err := Get(config)
	if err != nil {
		panic(err)
	}
	return req
}

func Post(config *Config) (*http.Request, error) {
	req, err := NewRequest("POST", config)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MustPost(config *Config) *http.Request {
	req, err := Post(config)
	if err != nil {
		panic(err)
	}
	return req
}

func Put(config *Config) (*http.Request, error) {
	req, err := NewRequest("PUT", config)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MustPut(config *Config) *http.Request {
	req, err := Put(config)
	if err != nil {
		panic(err)
	}
	return req
}

func Patch(config *Config) (*http.Request, error) {
	req, err := NewRequest("PATCH", config)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MustPatch(config *Config) *http.Request {
	req, err := Patch(config)
	if err != nil {
		panic(err)
	}
	return req
}

func Head(config *Config) (*http.Request, error) {
	req, err := NewRequest("HEAD", config)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MustHead(config *Config) *http.Request {
	req, err := Head(config)
	if err != nil {
		panic(err)
	}
	return req
}

func Options(config *Config) (*http.Request, error) {
	req, err := NewRequest("OPTIONS", config)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MustOptions(config *Config) *http.Request {
	req, err := Options(config)
	if err != nil {
		panic(err)
	}
	return req
}

func Delete(config *Config) (*http.Request, error) {
	req, err := NewRequest("DELETE", config)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MustDelete(config *Config) *http.Request {
	req, err := Delete(config)
	if err != nil {
		panic(err)
	}
	return req
}

func Trace(config *Config) (*http.Request, error) {
	req, err := NewRequest("TRACE", config)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MustTrace(config *Config) *http.Request {
	req, err := Trace(config)
	if err != nil {
		panic(err)
	}
	return req
}

func Connect(config *Config) (*http.Request, error) {
	req, err := NewRequest("CONNECT", config)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MustConnect(config *Config) *http.Request {
	req, err := Connect(config)
	if err != nil {
		panic(err)
	}
	return req
}
