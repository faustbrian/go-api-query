package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeClosedRejectsDuplicateMembers(t *testing.T) {
	t.Parallel()

	err := rejectDuplicateJSONMembers([]byte(`{"schema":"one","schema":"two"}`))
	if err == nil || !strings.Contains(err.Error(), "duplicate JSON member") {
		t.Fatalf("duplicate member error = %v", err)
	}
}

func TestIsolatedPostgresStorageRejectsVolumes(t *testing.T) {
	t.Parallel()

	storage := dockerStorage{Tmpfs: map[string]string{"/var/lib/postgresql": "rw,nosuid"}}
	if !isolatedPostgresStorage(storage) {
		t.Fatal("declared PostgreSQL tmpfs was rejected")
	}
	storage.Mounts = []dockerMount{{Type: "volume", Destination: "/var/lib/postgresql"}}
	if isolatedPostgresStorage(storage) {
		t.Fatal("PostgreSQL volume was accepted")
	}
}

func TestRemoveTreeHandlesReadOnlyModuleCacheDirectories(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "module-cache")
	nested := filepath.Join(root, "module@v1.0.0")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "source.go"), []byte("package source\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(nested, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := removeTree(root); err != nil {
		t.Fatal(err)
	}
	if exists(root) {
		t.Fatal("module cache root remains")
	}
}

func TestContainerCleanupTargetExistsBeforeStart(t *testing.T) {
	t.Parallel()

	cleanupTarget := ""
	defer func() {
		if recover() == nil {
			t.Fatal("container start did not fail")
		}
		if cleanupTarget != "owned-container" {
			t.Fatalf("cleanup target = %q", cleanupTarget)
		}
	}()
	startOwnedContainer("owned-container", &cleanupTarget, func() string {
		panic("start failed after create")
	})
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

func TestReceiptOnlyDiffRejectsSourceDrift(t *testing.T) {
	t.Parallel()

	if !receiptOnlyDiff([]byte(receiptPath + "\x00")) {
		t.Fatal("receipt-only diff was rejected")
	}
	if receiptOnlyDiff([]byte(receiptPath + "\x00go.mod\x00")) {
		t.Fatal("source drift was accepted")
	}
	if receiptOnlyDiff(nil) {
		t.Fatal("missing receipt change was accepted")
	}
}
