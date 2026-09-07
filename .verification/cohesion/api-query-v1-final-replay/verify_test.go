package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeClosedRejectsDuplicateMembers(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("duplicate member was accepted")
		}
	}()
	var value struct {
		Schema string `json:"schema"`
	}
	decodeClosed([]byte(`{"schema":"one","schema":"two"}`), &value)
}

func TestRejectDuplicateJSONMembersBoundsDepth(t *testing.T) {
	t.Parallel()

	data := []byte(strings.Repeat(`[`, 66) + `0` + strings.Repeat(`]`, 66))
	if err := rejectDuplicateJSONMembers(data); err == nil {
		t.Fatal("excess nesting was accepted")
	}
}

func TestModuleReceiptDecoderAcceptsGoCommandFields(t *testing.T) {
	t.Parallel()

	data := []byte(`{
        "Path":"github.com/faustbrian/go-api-query",
        "Version":"v1.1.0-replay.0",
        "Info":"/tmp/info",
        "GoMod":"/tmp/go.mod",
        "Zip":"/tmp/module.zip",
        "Dir":"/tmp/module",
        "Sum":"h1:module",
        "GoModSum":"h1:mod",
        "Origin":{"VCS":"git","URL":"https://example.invalid","Subdir":"","Hash":"abc","TagPrefix":"","TagSum":"","Ref":"refs/tags/v1.1.0","RepoSum":""},
        "Reuse":true
    }`)
	var receipt moduleDownload
	decodeClosed(data, &receipt)
	if receipt.Path != modulePath || receipt.Version != candidateVersion || !receipt.Reuse {
		t.Fatalf("decoded receipt = %#v", receipt)
	}
}

func TestWithEnvReplacesHostOverrides(t *testing.T) {
	t.Parallel()

	environment := withEnv([]string{"PATH=/bin", "GOFLAGS=-mod=vendor", "GONOPROXY=*"}, "GOFLAGS=", "GONOPROXY=none")
	encoded, err := json.Marshal(environment)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, "-mod=vendor") || strings.Contains(text, `GONOPROXY=*`) || !strings.Contains(text, `GOFLAGS=`) || !strings.Contains(text, `GONOPROXY=none`) {
		t.Fatalf("environment = %s", text)
	}
}
