package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/genai"
)

// Test-scoped model names derived from production maps. When the default
// alias changes (e.g. flash → flash-4.0), these update automatically.
var (
	testFlashName    = modelAliases["flash"]
	testFlashModelID = modelDefs[modelAliases["flash"]].ID
	testProName      = modelAliases["pro"]
	testProModelID   = modelDefs[modelAliases["pro"]].ID
)

func TestRunUsage(t *testing.T) {
	t.Run("no args returns error with usage", func(t *testing.T) {
		err := run(nil)
		if err == nil {
			t.Fatal("expected error for no args")
		}
		if !strings.Contains(err.Error(), "usage:") {
			t.Errorf("error = %q, want to contain 'usage:'", err)
		}
		if !strings.Contains(err.Error(), "meta") || !strings.Contains(err.Error(), "clean") {
			t.Error("usage should mention subcommands")
		}
	})

	t.Run("help returns nil", func(t *testing.T) {
		for _, arg := range []string{"help", "-h", "--help"} {
			if err := run([]string{arg}); err != nil {
				t.Errorf("run(%q) = %v, want nil", arg, err)
			}
		}
	})
}

func TestParseAndValidateFlags(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "test-key")

	tests := []struct {
		name    string
		args    []string
		setup   func(t *testing.T)
		wantErr string
		check   func(t *testing.T, opts *options)
	}{
		{
			name:    "positional args rejected",
			args:    []string{"-p", "hello", "world", "-o", "out.png"},
			wantErr: "unexpected arguments",
		},
		{
			name:    "missing prompt",
			args:    []string{"-o", "out.png"},
			wantErr: "usage",
		},
		{
			name:    "whitespace only prompt",
			args:    []string{"-p", "   ", "-o", "out.png"},
			wantErr: "usage",
		},
		{
			name:    "missing output",
			args:    []string{"-p", "a cat"},
			wantErr: "usage",
		},
		{
			name:    "unknown model",
			args:    []string{"-p", "a cat", "-o", "out.png", "-m", "turbo"},
			wantErr: "turbo",
		},
		{
			name:    "invalid ratio",
			args:    []string{"-p", "a cat", "-o", "out.png", "-r", "5:3"},
			wantErr: "5:3",
		},
		{
			name:    "invalid size value",
			args:    []string{"-p", "a cat", "-o", "out.png", "-m", "pro", "-z", "8k"},
			wantErr: "invalid size",
		},
		{
			name:    "size without pro",
			args:    []string{"-p", "a cat", "-o", "out.png", "-z", "2k"},
			wantErr: "pro model",
		},
		{
			name: "valid size normalized",
			args: []string{"-p", "a cat", "-o", "out.png", "-m", "pro", "-z", "2k"},
			check: func(t *testing.T, opts *options) {
				if opts.size != "2K" {
					t.Errorf("size = %q, want %q", opts.size, "2K")
				}
			},
		},
		{
			name:    "flash-2.5 input count exceeded",
			args:    []string{"-p", "a cat", "-o", "out.png", "-m", "flash-2.5", "-i", "a.png", "-i", "b.png", "-i", "c.png", "-i", "d.png"},
			wantErr: "supports up to 3 input images, got 4",
		},
		{
			name: "pro input count exceeded",
			args: func() []string {
				a := []string{"-p", "a cat", "-o", "out.png", "-m", "pro"}
				for i := 0; i < 15; i++ {
					a = append(a, "-i", "x.png")
				}
				return a
			}(),
			wantErr: "got 15",
		},
		{
			name:    "missing api key",
			args:    []string{"-p", "a cat", "-o", "out.png"},
			setup:   func(t *testing.T) { t.Setenv("GOOGLE_API_KEY", "") },
			wantErr: "GOOGLE_API_KEY",
		},
		{
			name:    "unsupported output extension",
			args:    []string{"-p", "a cat", "-o", "out.bmp"},
			wantErr: "output file must be .png",
		},
		{
			name:    "non-png image output rejected",
			args:    []string{"-p", "a cat", "-o", "out.jpg"},
			wantErr: "output file must be .png",
		},
		{
			name: "valid full flags with pro alias",
			args: []string{"-p", "a cat", "-o", "out.png", "-m", "pro", "-r", "16:9", "-z", "4k", "-f"},
			check: func(t *testing.T, opts *options) {
				if opts.prompt != "a cat" {
					t.Errorf("prompt = %q, want %q", opts.prompt, "a cat")
				}
				if opts.output != "out.png" {
					t.Errorf("output = %q, want %q", opts.output, "out.png")
				}
				if opts.model != testProName {
					t.Errorf("model = %q, want %q", opts.model, testProName)
				}
				if opts.modelID != testProModelID {
					t.Errorf("modelID = %q, want %q", opts.modelID, testProModelID)
				}
				if opts.ratio != "16:9" {
					t.Errorf("ratio = %q, want %q", opts.ratio, "16:9")
				}
				if opts.size != "4K" {
					t.Errorf("size = %q, want %q", opts.size, "4K")
				}
				if !opts.force {
					t.Error("force = false, want true")
				}
			},
		},
		{
			name: "flash alias resolves to current default",
			args: []string{"-p", "a cat", "-o", "out.png"},
			check: func(t *testing.T, opts *options) {
				if opts.model != testFlashName {
					t.Errorf("model = %q, want %q", opts.model, testFlashName)
				}
				if opts.modelID != testFlashModelID {
					t.Errorf("modelID = %q, want %q", opts.modelID, testFlashModelID)
				}
			},
		},
		{
			name: "pinned flash-2.5",
			args: []string{"-p", "a cat", "-o", "out.png", "-m", "flash-2.5"},
			check: func(t *testing.T, opts *options) {
				if opts.model != "flash-2.5" {
					t.Errorf("model = %q, want %q", opts.model, "flash-2.5")
				}
				if opts.modelID != "gemini-2.5-flash-image" {
					t.Errorf("modelID = %q, want %q", opts.modelID, "gemini-2.5-flash-image")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			opts, err := parseAndValidateFlags(tt.args)
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
			if tt.check != nil {
				tt.check(t, opts)
			}
		})
	}
}

func TestValidatePaths(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *options
		wantErr string
	}{
		{
			name: "output dir missing",
			setup: func(t *testing.T) *options {
				return &options{output: "/nonexistent_dir_xyz/out.png"}
			},
			wantErr: "does not exist",
		},
		{
			name: "session save collides with source",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				sess := filepath.Join(dir, "out.session.json")
				os.WriteFile(sess, []byte("{}"), 0644)
				return &options{output: filepath.Join(dir, "out.png"), session: sess}
			},
			wantErr: "collides with -s source",
		},
		{
			name: "session save collides with source force",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				sess := filepath.Join(dir, "out.session.json")
				os.WriteFile(sess, []byte("{}"), 0644)
				return &options{output: filepath.Join(dir, "out.png"), session: sess, force: true}
			},
		},
		{
			name: "output exists without force",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				p := filepath.Join(dir, "out.png")
				os.WriteFile(p, []byte("x"), 0644)
				return &options{output: p}
			},
			wantErr: "already exists",
		},
		{
			name: "output exists with force",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				p := filepath.Join(dir, "out.png")
				os.WriteFile(p, []byte("x"), 0644)
				return &options{output: p, force: true}
			},
		},
		{
			name: "session file collision on fresh run",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				outPath := filepath.Join(dir, "out.png")
				os.WriteFile(sessionPath(outPath), []byte("{}"), 0644)
				return &options{output: outPath}
			},
			wantErr: "already exists",
		},
		{
			name: "session file collision with force",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				outPath := filepath.Join(dir, "out.png")
				os.WriteFile(sessionPath(outPath), []byte("{}"), 0644)
				return &options{output: outPath, force: true}
			},
		},
		{
			name: "input file missing",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				return &options{
					output: filepath.Join(dir, "out.png"),
					inputs: stringSlice{"/nonexistent_xyz/image.png"},
				}
			},
			wantErr: "does not exist",
		},
		{
			name: "input file bad extension",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				badFile := filepath.Join(dir, "image.bmp")
				os.WriteFile(badFile, []byte("x"), 0644)
				return &options{
					output: filepath.Join(dir, "out.png"),
					inputs: stringSlice{badFile},
				}
			},
			wantErr: "unsupported extension",
		},
		{
			name: "input file over 7MB",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				bigFile := filepath.Join(dir, "huge.png")
				os.WriteFile(bigFile, make([]byte, 8*1024*1024), 0644)
				return &options{
					output: filepath.Join(dir, "out.png"),
					inputs: stringSlice{bigFile},
				}
			},
			wantErr: "exceeds 7 MB",
		},
		{
			name: "session collision checked even with session flag",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				outPath := filepath.Join(dir, "out.png")
				os.WriteFile(sessionPath(outPath), []byte("{}"), 0644)
				return &options{output: outPath, session: filepath.Join(dir, "other.json")}
			},
			wantErr: "already exists",
		},
		{
			name: "all valid",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				inputFile := filepath.Join(dir, "input.png")
				os.WriteFile(inputFile, []byte("x"), 0644)
				return &options{
					output: filepath.Join(dir, "out.png"),
					inputs: stringSlice{inputFile},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.setup(t)
			err := validatePaths(opts)
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
		})
	}
}

func TestLoadSession(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string // returns session file path
		model   string
		wantErr string
		wantLen int
	}{
		{
			name: "file missing",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing.json")
			},
			model:   "flash",
			wantErr: "failed to read",
		},
		{
			name: "invalid json",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "bad.json")
				os.WriteFile(p, []byte("{invalid"), 0644)
				return p
			},
			model:   "flash",
			wantErr: "failed to parse",
		},
		{
			name: "exact pinned match",
			setup: func(t *testing.T) string {
				sess := sessionData{
					Model: "flash-3.1",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hello"}}},
					},
				}
				data, _ := json.Marshal(sess)
				p := filepath.Join(t.TempDir(), "session.json")
				os.WriteFile(p, data, 0644)
				return p
			},
			model:   "flash-3.1",
			wantLen: 1,
		},
		{
			name: "pinned model mismatch",
			setup: func(t *testing.T) string {
				sess := sessionData{Model: "pro-3.0", History: []*genai.Content{}}
				data, _ := json.Marshal(sess)
				p := filepath.Join(t.TempDir(), "session.json")
				os.WriteFile(p, data, 0644)
				return p
			},
			model:   "flash-3.1",
			wantErr: "pro-3.0",
		},
		{
			name: "legacy alias same family allowed",
			setup: func(t *testing.T) string {
				sess := sessionData{
					Model: "flash",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hello"}}},
					},
				}
				data, _ := json.Marshal(sess)
				p := filepath.Join(t.TempDir(), "session.json")
				os.WriteFile(p, data, 0644)
				return p
			},
			model:   "flash-3.1",
			wantLen: 1,
		},
		{
			name: "legacy alias different family rejected",
			setup: func(t *testing.T) string {
				sess := sessionData{Model: "flash", History: []*genai.Content{}}
				data, _ := json.Marshal(sess)
				p := filepath.Join(t.TempDir(), "session.json")
				os.WriteFile(p, data, 0644)
				return p
			},
			model:   "pro-3.0",
			wantErr: "flash",
		},
		{
			name: "empty model skips check",
			setup: func(t *testing.T) string {
				sess := sessionData{
					Model: "",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hello"}}},
					},
				}
				data, _ := json.Marshal(sess)
				p := filepath.Join(t.TempDir(), "session.json")
				os.WriteFile(p, data, 0644)
				return p
			},
			model:   "pro-3.0",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			history, err := loadSession(path, tt.model)
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
			if len(history) != tt.wantLen {
				t.Fatalf("history length = %d, want %d", len(history), tt.wantLen)
			}
		})
	}
}

func TestExtractResult(t *testing.T) {
	imgBytes := []byte("fake-image-data")

	tests := []struct {
		name     string
		result   *genai.GenerateContentResponse
		wantErr  string
		wantText string
		wantImg  []byte
	}{
		{
			name:    "nil result",
			result:  nil,
			wantErr: "no response from model",
		},
		{
			name:    "empty candidates",
			result:  &genai.GenerateContentResponse{},
			wantErr: "no response from model",
		},
		{
			name: "block reason",
			result: &genai.GenerateContentResponse{
				PromptFeedback: &genai.GenerateContentResponsePromptFeedback{
					BlockReason: "SAFETY",
				},
			},
			wantErr: "prompt blocked",
		},
		{
			name: "nil content with finish reason",
			result: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: nil, FinishReason: "SAFETY"},
				},
			},
			wantErr: "SAFETY",
		},
		{
			name: "nil content without finish reason",
			result: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: nil},
				},
			},
			wantErr: "unknown",
		},
		{
			name: "text only no image",
			result: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "hello"},
						},
					}},
				},
			},
			wantErr: "no image data",
		},
		{
			name: "thought parts excluded",
			result: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "thinking...", Thought: true},
							{Text: "visible"},
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: imgBytes}},
						},
					}},
				},
			},
			wantText: "visible",
			wantImg:  imgBytes,
		},
		{
			name: "text and image",
			result: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "description"},
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: imgBytes}},
						},
					}},
				},
			},
			wantText: "description",
			wantImg:  imgBytes,
		},
		{
			name: "multiple text parts joined",
			result: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "line one"},
							{Text: "line two"},
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: imgBytes}},
						},
					}},
				},
			},
			wantText: "line one\nline two",
			wantImg:  imgBytes,
		},
		{
			name: "multiple images returns first",
			result: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{
						Parts: []*genai.Part{
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("first")}},
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("second")}},
						},
					}},
				},
			},
			wantImg: []byte("first"),
		},
		{
			name: "nil parts in slice",
			result: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{
						Parts: []*genai.Part{
							nil,
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: imgBytes}},
						},
					}},
				},
			},
			wantImg: imgBytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, img, err := extractResult(tt.result)
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
			if text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}
			if !bytes.Equal(img, tt.wantImg) {
				t.Errorf("image data mismatch: got %d bytes, want %d bytes", len(img), len(tt.wantImg))
			}
		})
	}
}

