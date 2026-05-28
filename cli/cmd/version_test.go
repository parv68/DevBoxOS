package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(&buf, r)
	}()

	fn()

	w.Close()
	wg.Wait()
	os.Stdout = old

	return buf.String()
}

func TestVersionCmd_Output(t *testing.T) {
	origVersion := version
	origCommit := commit
	origDate := date
	version = "1.0.0-test"
	commit = "abc123"
	date = "2026-01-01"
	defer func() {
		version = origVersion
		commit = origCommit
		date = origDate
	}()

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"version"})
		rootCmd.Execute()
	})

	expected := fmt.Sprintf("DevBoxOS %s (commit: %s, built: %s)", version, commit, date)
	if !strings.Contains(output, expected) {
		t.Errorf("expected output containing %q, got %q", expected, output)
	}
}

func TestVersionCmd_DefaultValues(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"version"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "DevBoxOS") {
		t.Errorf("output should contain DevBoxOS, got %q", output)
	}
}
