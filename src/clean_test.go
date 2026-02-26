package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/genai"
)

// writeSessionFile is a test helper that writes a sessionData JSON file and returns its path.
func writeSessionFile(t *testing.T, dir, name string, sess sessionData) string {
	t.Helper()
	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestValidateSessionFile(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantErr   string
		wantModel string
		wantTurns int
	}{
		{
			name: "valid flash session",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: "flash",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "a cat"}}},
						{Role: "model", Parts: []*genai.Part{{Text: "here"}}},
					},
				})
			},
			wantModel: "flash",
			wantTurns: 1,
		},
		{
			name: "valid pro session",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: "pro",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "a"}}},
						{Role: "model", Parts: []*genai.Part{{Text: "b"}}},
						{Role: "user", Parts: []*genai.Part{{Text: "c"}}},
						{Role: "model", Parts: []*genai.Part{{Text: "d"}}},
					},
				})
			},
			wantModel: "pro",
			wantTurns: 2,
		},
		{
			name: "legacy session empty model",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model:   "",
					History: []*genai.Content{},
				})
			},
			wantModel: "",
			wantTurns: 0,
		},
		{
			name: "odd history rounds up turns",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: "flash",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "a"}}},
						{Role: "model", Parts: []*genai.Part{{Text: "b"}}},
						{Role: "user", Parts: []*genai.Part{{Text: "c"}}},
					},
				})
			},
			wantModel: "flash",
			wantTurns: 2,
		},
		{
			name: "invalid JSON",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "bad.session.json")
				os.WriteFile(p, []byte("{not json"), 0644)
				return p
			},
			wantErr: "not a banana session",
		},
		{
			name: "wrong structure rejects unknown fields",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "wrong.session.json")
				os.WriteFile(p, []byte(`{"foo":"bar"}`), 0644)
				return p
			},
			wantErr: "not a banana session",
		},
		{
			name: "extra fields rejected",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "extra.session.json")
				os.WriteFile(p, []byte(`{"model":"flash","history":[],"extra":true}`), 0644)
				return p
			},
			wantErr: "not a banana session",
		},
		{
			name: "unknown model value",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model:   "turbo",
					History: []*genai.Content{},
				})
			},
			wantErr: "unknown model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			si, err := validateSessionFile(path)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if si.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", si.Model, tt.wantModel)
			}
			if si.Turns != tt.wantTurns {
				t.Errorf("Turns = %d, want %d", si.Turns, tt.wantTurns)
			}
			if si.Size <= 0 {
				t.Errorf("Size = %d, want > 0", si.Size)
			}
		})
	}
}

func TestRunClean(t *testing.T) {
	// Helper to create a populated test directory.
	makeDir := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		writeSessionFile(t, dir, "a.session.json", sessionData{Model: "flash", History: []*genai.Content{
			{Role: "user", Parts: []*genai.Part{{Text: "hi"}}},
			{Role: "model", Parts: []*genai.Part{{Text: "hey"}}},
		}})
		writeSessionFile(t, dir, "b.session.json", sessionData{Model: "pro", History: []*genai.Content{}})
		return dir
	}

	tests := []struct {
		name       string
		args       func(t *testing.T) []string
		wantErr    string
		checkAfter func(t *testing.T, dir string)
	}{
		{
			name: "dry run lists without deleting",
			args: func(t *testing.T) []string {
				dir := makeDir(t)
				return []string{dir}
			},
			checkAfter: func(t *testing.T, dir string) {
				// Both files should still exist
				for _, name := range []string{"a.session.json", "b.session.json"} {
					if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
						t.Errorf("file %s was unexpectedly deleted in dry run", name)
					}
				}
			},
		},
		{
			name: "force deletes validated files",
			args: func(t *testing.T) []string {
				dir := makeDir(t)
				return []string{"-f", dir}
			},
			checkAfter: func(t *testing.T, dir string) {
				for _, name := range []string{"a.session.json", "b.session.json"} {
					if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
						t.Errorf("file %s was not deleted with -f", name)
					}
				}
			},
		},
		{
			name: "skips non-session files",
			args: func(t *testing.T) []string {
				dir := makeDir(t)
				// Add a non-session JSON file and a regular file
				os.WriteFile(filepath.Join(dir, "notes.json"), []byte(`{}`), 0644)
				os.WriteFile(filepath.Join(dir, "image.png"), []byte("img"), 0644)
				return []string{"-f", dir}
			},
			checkAfter: func(t *testing.T, dir string) {
				// Non-session files should survive
				for _, name := range []string{"notes.json", "image.png"} {
					if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
						t.Errorf("non-session file %s was unexpectedly deleted", name)
					}
				}
				// Session files should be gone
				for _, name := range []string{"a.session.json", "b.session.json"} {
					if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
						t.Errorf("session file %s was not deleted", name)
					}
				}
			},
		},
		{
			name: "skips invalid session files",
			args: func(t *testing.T) []string {
				dir := makeDir(t)
				os.WriteFile(filepath.Join(dir, "bad.session.json"), []byte("{corrupt"), 0644)
				return []string{"-f", dir}
			},
			checkAfter: func(t *testing.T, dir string) {
				// Invalid session file should survive
				if _, err := os.Stat(filepath.Join(dir, "bad.session.json")); err != nil {
					t.Error("invalid session file was unexpectedly deleted")
				}
				// Valid ones should be gone
				for _, name := range []string{"a.session.json", "b.session.json"} {
					if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
						t.Errorf("valid session file %s was not deleted", name)
					}
				}
			},
		},
		{
			name: "does not recurse into subdirectories",
			args: func(t *testing.T) []string {
				dir := makeDir(t)
				sub := filepath.Join(dir, "nested")
				os.Mkdir(sub, 0755)
				writeSessionFile(t, sub, "deep.session.json", sessionData{Model: "flash", History: []*genai.Content{}})
				return []string{"-f", dir}
			},
			checkAfter: func(t *testing.T, dir string) {
				// Top-level session files should be deleted
				for _, name := range []string{"a.session.json", "b.session.json"} {
					if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
						t.Errorf("top-level file %s was not deleted", name)
					}
				}
				// Nested session file must survive
				nested := filepath.Join(dir, "nested", "deep.session.json")
				if _, err := os.Stat(nested); err != nil {
					t.Error("nested session file was unexpectedly deleted")
				}
			},
		},
		{
			name: "missing directory argument",
			args: func(t *testing.T) []string {
				return []string{}
			},
			wantErr: "usage:",
		},
		{
			name: "flag after directory gives targeted hint",
			args: func(t *testing.T) []string {
				dir := makeDir(t)
				return []string{dir, "-f"}
			},
			wantErr: "flag -f must appear before the directory",
		},
		{
			name: "invalid directory",
			args: func(t *testing.T) []string {
				return []string{fmt.Sprintf("/nonexistent_%d", os.Getpid())}
			},
			wantErr: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args(t)
			err := runClean(args)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkAfter != nil {
				// Extract directory from args (last element)
				dir := args[len(args)-1]
				tt.checkAfter(t, dir)
			}
		})
	}
}
