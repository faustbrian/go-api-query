package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDescriptorCountUsesPathResolvedLsof(t *testing.T) {
	directory := t.TempDir()
	marker := filepath.Join(directory, "called")
	executable := filepath.Join(directory, "lsof")
	content := "#!/bin/sh\n: > '" + marker + "'\nprintf 'p123\\nnfirst\\nnsecond\\n'\n"
	if err := os.WriteFile(executable, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)

	if count := descriptorCount(context.Background()); count != 2 {
		t.Fatalf("descriptorCount() = %d, want 2", count)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("PATH-resolved lsof was not called: %v", err)
	}
}
