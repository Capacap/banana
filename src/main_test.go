package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/genai"
)

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
			wantErr: "pro",
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
			name:    "flash input count exceeded",
			args:    []string{"-p", "a cat", "-o", "out.png", "-i", "a.png", "-i", "b.png", "-i", "c.png", "-i", "d.png"},
			wantErr: "use -m pro",
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
			wantErr: "unsupported extension",
		},
		{
			name: "valid full flags",
			args: []string{"-p", "a cat", "-o", "out.png", "-m", "pro", "-r", "16:9", "-z", "4k", "-f"},
			check: func(t *testing.T, opts *options) {
				if opts.prompt != "a cat" {
					t.Errorf("prompt = %q, want %q", opts.prompt, "a cat")
				}
				if opts.output != "out.png" {
					t.Errorf("output = %q, want %q", opts.output, "out.png")
				}
				if opts.model != "pro" {
					t.Errorf("model = %q, want %q", opts.model, "pro")
				}
				if opts.modelID != "gemini-3-pro-image-preview" {
					t.Errorf("modelID = %q, want %q", opts.modelID, "gemini-3-pro-image-preview")
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
			name: "output and session same path",
			setup: func(t *testing.T) *options {
				dir := t.TempDir()
				p := filepath.Join(dir, "file.png")
				return &options{output: p, session: p}
			},
			wantErr: "-o and -s must not point to the same file",
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
			wantErr: "failed to read session",
		},
		{
			name: "invalid json",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "bad.json")
				os.WriteFile(p, []byte("{invalid"), 0644)
				return p
			},
			model:   "flash",
			wantErr: "failed to parse session",
		},
		{
			name: "valid new format",
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
			model:   "flash",
			wantLen: 1,
		},
		{
			name: "model mismatch",
			setup: func(t *testing.T) string {
				sess := sessionData{Model: "pro", History: []*genai.Content{}}
				data, _ := json.Marshal(sess)
				p := filepath.Join(t.TempDir(), "session.json")
				os.WriteFile(p, data, 0644)
				return p
			},
			model:   "flash",
			wantErr: "pro",
		},
		{
			name: "legacy format empty model",
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
			model:   "pro",
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

func TestBuildMetadata(t *testing.T) {
	tests := []struct {
		name        string
		opts        *options
		history     []*genai.Content
		wantPrompts int
		check       func(t *testing.T, meta imageMetadata)
	}{
		{
			name: "basic text extraction",
			opts: &options{model: "flash", modelID: "gemini-2.5-flash-image", ratio: "1:1"},
			history: []*genai.Content{
				{Role: "user", Parts: []*genai.Part{{Text: "a cat"}}},
				{Role: "model", Parts: []*genai.Part{{Text: "here it is"}}},
			},
			wantPrompts: 2,
			check: func(t *testing.T, meta imageMetadata) {
				if meta.Prompts[0].Role != "user" || meta.Prompts[0].Text != "a cat" {
					t.Errorf("prompt 0 = %+v", meta.Prompts[0])
				}
				if meta.Prompts[1].Role != "model" || meta.Prompts[1].Text != "here it is" {
					t.Errorf("prompt 1 = %+v", meta.Prompts[1])
				}
			},
		},
		{
			name: "inline data excluded",
			opts: &options{model: "pro", modelID: "gemini-3-pro-image-preview", ratio: "16:9"},
			history: []*genai.Content{
				{Role: "user", Parts: []*genai.Part{
					{Text: "edit this"},
					{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("imgdata")}},
				}},
				{Role: "model", Parts: []*genai.Part{
					{Text: "done"},
					{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("result")}},
				}},
			},
			wantPrompts: 2,
			check: func(t *testing.T, meta imageMetadata) {
				if meta.Prompts[0].Text != "edit this" {
					t.Errorf("user text = %q, want %q", meta.Prompts[0].Text, "edit this")
				}
			},
		},
		{
			name: "thought parts excluded",
			opts: &options{model: "flash", modelID: "gemini-2.5-flash-image", ratio: "1:1"},
			history: []*genai.Content{
				{Role: "user", Parts: []*genai.Part{{Text: "hello"}}},
				{Role: "model", Parts: []*genai.Part{
					{Text: "thinking...", Thought: true},
					{Text: "visible"},
				}},
			},
			wantPrompts: 2,
			check: func(t *testing.T, meta imageMetadata) {
				if meta.Prompts[1].Text != "visible" {
					t.Errorf("model text = %q, want %q", meta.Prompts[1].Text, "visible")
				}
			},
		},
		{
			name: "inputs populated",
			opts: &options{
				model: "flash", modelID: "gemini-2.5-flash-image", ratio: "1:1",
				inputs: stringSlice{"ref.png", "bg.jpg"},
			},
			history:     []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: "go"}}}},
			wantPrompts: 1,
			check: func(t *testing.T, meta imageMetadata) {
				if len(meta.Inputs) != 2 || meta.Inputs[0] != "ref.png" {
					t.Errorf("inputs = %v", meta.Inputs)
				}
			},
		},
		{
			name: "nil content in history skipped",
			opts: &options{model: "flash", modelID: "gemini-2.5-flash-image", ratio: "1:1"},
			history: []*genai.Content{
				nil,
				{Role: "user", Parts: []*genai.Part{{Text: "hello"}}},
			},
			wantPrompts: 1,
		},
		{
			name: "fields populated from opts",
			opts: &options{model: "pro", modelID: "gemini-3-pro-image-preview", ratio: "3:2", size: "4K"},
			history: []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: "x"}}}},
			wantPrompts: 1,
			check: func(t *testing.T, meta imageMetadata) {
				if meta.Model != "pro" {
					t.Errorf("model = %q", meta.Model)
				}
				if meta.ModelID != "gemini-3-pro-image-preview" {
					t.Errorf("model_id = %q", meta.ModelID)
				}
				if meta.Ratio != "3:2" {
					t.Errorf("ratio = %q", meta.Ratio)
				}
				if meta.Size != "4K" {
					t.Errorf("size = %q", meta.Size)
				}
				if meta.Timestamp == "" {
					t.Error("timestamp is empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := buildMetadata(tt.opts, tt.history)
			if len(meta.Prompts) != tt.wantPrompts {
				t.Fatalf("prompts count = %d, want %d", len(meta.Prompts), tt.wantPrompts)
			}
			if tt.check != nil {
				tt.check(t, meta)
			}
		})
	}
}

func TestRunMeta(t *testing.T) {
	t.Run("valid embedded metadata", func(t *testing.T) {
		dir := t.TempDir()
		png := minimalPNG()
		meta := imageMetadata{
			Model:     "flash",
			ModelID:   "gemini-2.5-flash-image",
			Ratio:     "1:1",
			Timestamp: "2026-02-26T12:00:00Z",
			Prompts:   []promptEntry{{Role: "user", Text: "a cat"}},
		}
		jsonBytes, _ := json.Marshal(meta)
		embedded, err := pngSetText(png, "banana", string(jsonBytes))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "test.png")
		os.WriteFile(path, embedded, 0644)

		if err := runMeta([]string{path}); err != nil {
			t.Fatalf("runMeta: %v", err)
		}
	})

	t.Run("no metadata", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "plain.png")
		os.WriteFile(path, minimalPNG(), 0644)

		err := runMeta([]string{path})
		if err == nil {
			t.Fatal("expected error for PNG without metadata")
		}
		if !strings.Contains(err.Error(), "no banana metadata") {
			t.Fatalf("error = %q", err)
		}
	})

	t.Run("non-PNG file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "fake.png")
		os.WriteFile(path, []byte("not a png"), 0644)

		err := runMeta([]string{path})
		if err == nil {
			t.Fatal("expected error for non-PNG file")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		err := runMeta([]string{"/nonexistent/file.png"})
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("no args", func(t *testing.T) {
		err := runMeta(nil)
		if err == nil {
			t.Fatal("expected error for no args")
		}
		if !strings.Contains(err.Error(), "usage") {
			t.Fatalf("error = %q", err)
		}
	})
}

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
		name       string
		setup      func(t *testing.T) string
		wantErr    string
		wantModel  string
		wantTurns  int
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
		name      string
		args      func(t *testing.T) []string
		wantErr   string
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
