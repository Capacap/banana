package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/genai"
)

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
			opts: &options{model: "flash-3.1", modelID: "gemini-3.1-flash-image-preview", ratio: "1:1"},
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
			opts: &options{model: "pro-3.0", modelID: "gemini-3-pro-image-preview", ratio: "16:9"},
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
			opts: &options{model: "flash-3.1", modelID: "gemini-3.1-flash-image-preview", ratio: "1:1"},
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
			name: "inputs stores basenames only",
			opts: &options{
				model: "flash-3.1", modelID: "gemini-3.1-flash-image-preview", ratio: "1:1",
				inputs: stringSlice{"/home/user/images/ref.png", "../assets/bg.jpg"},
			},
			history:     []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: "go"}}}},
			wantPrompts: 1,
			check: func(t *testing.T, meta imageMetadata) {
				want := []string{"ref.png", "bg.jpg"}
				if len(meta.Inputs) != len(want) {
					t.Fatalf("inputs = %v, want %v", meta.Inputs, want)
				}
				for i := range want {
					if meta.Inputs[i] != want[i] {
						t.Errorf("inputs[%d] = %q, want %q", i, meta.Inputs[i], want[i])
					}
				}
			},
		},
		{
			name: "session stores basename only",
			opts: &options{
				model: "flash-3.1", modelID: "gemini-3.1-flash-image-preview", ratio: "1:1",
				session: "/home/user/work/cat.session.json",
			},
			history:     []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: "go"}}}},
			wantPrompts: 1,
			check: func(t *testing.T, meta imageMetadata) {
				if meta.Session != "cat.session.json" {
					t.Errorf("session = %q, want %q", meta.Session, "cat.session.json")
				}
			},
		},
		{
			name: "no session when flag absent",
			opts: &options{model: "flash-3.1", modelID: "gemini-3.1-flash-image-preview", ratio: "1:1"},
			history:     []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: "go"}}}},
			wantPrompts: 1,
			check: func(t *testing.T, meta imageMetadata) {
				if meta.Session != "" {
					t.Errorf("session = %q, want empty", meta.Session)
				}
			},
		},
		{
			name: "nil content in history skipped",
			opts: &options{model: "flash-3.1", modelID: "gemini-3.1-flash-image-preview", ratio: "1:1"},
			history: []*genai.Content{
				nil,
				{Role: "user", Parts: []*genai.Part{{Text: "hello"}}},
			},
			wantPrompts: 1,
		},
		{
			name: "fields populated from opts",
			opts: &options{model: "pro-3.0", modelID: "gemini-3-pro-image-preview", ratio: "3:2", size: "4K"},
			history: []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: "x"}}}},
			wantPrompts: 1,
			check: func(t *testing.T, meta imageMetadata) {
				if meta.Model != "pro-3.0" {
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
				if meta.Version != metadataVersion {
					t.Errorf("version = %d, want %d", meta.Version, metadataVersion)
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
			Version:   metadataVersion,
			Model:     "flash-3.1",
			ModelID:   "gemini-3.1-flash-image-preview",
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
		if !strings.Contains(err.Error(), "not a PNG file") {
			t.Fatalf("error = %q, want mention of 'not a PNG file'", err)
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
