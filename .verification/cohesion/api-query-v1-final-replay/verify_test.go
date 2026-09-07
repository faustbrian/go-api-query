package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecodeClosedRejectsDuplicateMembers(t *testing.T) {
	t.Parallel()

	err := rejectDuplicateJSONMembers([]byte(`{"schema":"one","schema":"two"}`))
	if err == nil || !strings.Contains(err.Error(), "duplicate JSON member") {
		t.Fatalf("duplicate member error = %v", err)
	}
}

func TestDeterministicModuleZipIgnoresTarEntryOrderAndMetadata(t *testing.T) {
	t.Parallel()

	firstTar := testTar(t, []string{"b.txt", "a.txt"}, time.Unix(1, 0), 0o664)
	secondTar := testTar(t, []string{"a.txt", "b.txt"}, time.Unix(2, 0), 0o600)
	modified := time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC)
	firstPath := filepath.Join(t.TempDir(), "first.zip")
	secondPath := filepath.Join(t.TempDir(), "second.zip")
	if err := writeDeterministicModuleZip(firstPath, "example.invalid/module@v1.0.0/", modified, firstTar); err != nil {
		t.Fatal(err)
	}
	if err := writeDeterministicModuleZip(secondPath, "example.invalid/module@v1.0.0/", modified, secondTar); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("module zip changed with tar ordering or metadata")
	}
}

func TestDeterministicModuleZipPreservesExecutableSemantics(t *testing.T) {
	t.Parallel()

	zipPath := filepath.Join(t.TempDir(), "module.zip")
	archive := testTar(t, []string{"script.sh"}, time.Unix(1, 0), 0o775)
	if err := writeDeterministicModuleZip(zipPath, "example.invalid/module@v1.0.0/", time.Unix(2, 0), archive); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 1 || reader.File[0].Mode().Perm() != 0o755 {
		t.Fatalf("module zip mode = %v", reader.File[0].Mode())
	}
}

func TestDeterministicModuleZipAcceptsGitGlobalPAXMetadata(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	if err := writer.WriteHeader(&tar.Header{
		Name:       "pax_global_header",
		Typeflag:   tar.TypeXGlobalHeader,
		PAXRecords: map[string]string{"comment": "source commit identity"},
	}); err != nil {
		t.Fatal(err)
	}
	data := []byte("content\n")
	if err := writer.WriteHeader(&tar.Header{Name: "file.txt", Mode: 0o644, Size: int64(len(data))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "module.zip")
	if err := writeDeterministicModuleZip(
		zipPath,
		"example.invalid/module@v1.0.0/",
		time.Unix(2, 0),
		buffer.Bytes(),
	); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 1 || reader.File[0].Name != "example.invalid/module@v1.0.0/file.txt" || reader.File[0].Mode().Perm() != 0o644 {
		t.Fatalf("module zip entries = %+v", reader.File)
	}
	file, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("module zip content = %q", got)
	}
}

func TestDeterministicModuleZipRejectsOversizedRawArchive(t *testing.T) {
	t.Parallel()

	archive := testTar(t, []string{"file.txt"}, time.Unix(1, 0), 0o644)
	archive = append(archive, make([]byte, maximumBytes+1-len(archive))...)
	err := writeDeterministicModuleZip(
		filepath.Join(t.TempDir(), "module.zip"),
		"example.invalid/module@v1.0.0/",
		time.Unix(2, 0),
		archive,
	)
	if err == nil || err.Error() != "source archive exceeds bound" {
		t.Fatalf("oversized archive error = %v", err)
	}
}

func testTar(t *testing.T, names []string, modified time.Time, mode int64) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for _, name := range names {
		data := []byte(name + "\n")
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data)), ModTime: modified}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
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
