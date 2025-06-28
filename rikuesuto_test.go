package rikuesuto

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	pastebin := "https://pastebin.com/raw/seCnUfhr"
	webhook := "https://discord.com/api/webhooks/0/A"
	messages := "https://discord.com/api/v9/channels/0/messages"
	token := "token"
	filename := "hello.txt"
	client := &http.Client{}

	req := MustGet(&Config{URL: pastebin})
	s, res := MustDoReadString(client, req)
	t.Logf("GET (%d): %s", res.StatusCode, s)

	req = MustPost(&Config{
		URL:  webhook,
		JSON: map[string]interface{}{"content": "hello"},
	})
	s, res = MustDoReadString(client, req)
	t.Logf("JSON POST (%d): %s", res.StatusCode, s)

	req = MustPost(&Config{
		URL:    messages,
		JSON:   map[string]interface{}{"content": "hello"},
		Header: map[string][]string{"authorization": {token}},
	})
	s, res = MustDoReadString(client, req)
	t.Logf("JSON POST with Header (%d): %s", res.StatusCode, s)

	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	req = MustPost(&Config{
		URL: webhook,
		Multipart: &MultipartData{
			Files:    map[string]*os.File{filename: file},
			Fields:   map[string]io.Reader{"content": strings.NewReader("hello")},
			Boundary: "END_OF_PART",
		},
	})
	s, res = MustDoReadString(client, req)
	t.Logf("Multipart POST with Header (%d): %s", res.StatusCode, s)
}
