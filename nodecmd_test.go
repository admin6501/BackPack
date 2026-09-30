package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNodeRequestIsReadFromStdinWithSizeLimit(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(`{"op":"hello"}`))
	got, err := readNodeRequest(strings.NewReader(encoded + "\n"))
	if err != nil || string(got) != `{"op":"hello"}` {
		t.Fatalf("read request = %q, %v", got, err)
	}
	if _, err := readNodeRequest(strings.NewReader(strings.Repeat("A", (16<<20)+1))); err == nil {
		t.Fatal("unbounded request accepted")
	}
}
