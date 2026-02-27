package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/genai"
)

func TestAnalyzeSession(t *testing.T) {
	flashDef := modelDefs[testFlashName]
	flash31Def := modelDefs["flash-3.1"]
	proDef := modelDefs[testProName]

	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		wantErr     string
		wantModel   string
		wantTurns   int
		wantImages  int
		wantInput   float64
		wantOutput  float64
		wantImage   float64
		wantTotal   float64
		noUsage     bool
	}{
		{
			name: "session with usage data",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Write raw JSON to include usage field
				data := fmt.Sprintf(`{"model":%q,"history":[{"role":"user","parts":[{"text":"a cat"}]},{"role":"model","parts":[{"inlineData":{"mimeType":"image/png","data":"aW1n"}}]}],"usage":{"prompt_tokens":1000,"candidate_tokens":200,"total_tokens":1200}}`, testFlashName)
				p := filepath.Join(dir, "test.session.json")
				os.WriteFile(p, []byte(data), 0644)
				return p
			},
			wantModel:  testFlashName,
			wantTurns:  1,
			wantImages: 1,
			wantInput:  float64(1000) * flashDef.InputPerMTok / 1_000_000,
			wantOutput: float64(200) * flashDef.OutputPerMTok / 1_000_000,
			wantImage:  flashDef.ImagePrices["1K"],
			wantTotal:  float64(1000)*flashDef.InputPerMTok/1_000_000 + float64(200)*flashDef.OutputPerMTok/1_000_000 + flashDef.ImagePrices["1K"],
		},
		{
			name: "session without usage data (legacy)",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: testFlashName,
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "a cat"}}},
						{Role: "model", Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img")}}}},
					},
				})
			},
			wantModel:  testFlashName,
			wantTurns:  1,
			wantImages: 1,
			wantImage:  flashDef.ImagePrices["1K"],
			wantTotal:  flashDef.ImagePrices["1K"],
			noUsage:    true,
		},
		{
			name: "pro session multiple images",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: testProName,
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "landscape"}}},
						{Role: "model", Parts: []*genai.Part{
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img1")}},
						}},
						{Role: "user", Parts: []*genai.Part{{Text: "add mountains"}}},
						{Role: "model", Parts: []*genai.Part{
							{Text: "here"},
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img2")}},
						}},
					},
				})
			},
			wantModel:  testProName,
			wantTurns:  2,
			wantImages: 2,
			wantImage:  2 * proDef.ImagePrices["1K"],
			wantTotal:  2 * proDef.ImagePrices["1K"],
			noUsage:    true,
		},
		{
			name: "legacy session empty model",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: "",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hi"}}},
						{Role: "model", Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img")}}}},
					},
				})
			},
			wantModel:  "",
			wantTurns:  1,
			wantImages: 1,
			wantTotal:  0, // unknown pricing
			noUsage:    true,
		},
		{
			name: "alias model name resolves",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: "flash",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hi"}}},
						{Role: "model", Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img")}}}},
					},
				})
			},
			wantModel:  testFlashName,
			wantTurns:  1,
			wantImages: 1,
			wantImage:  flashDef.ImagePrices["1K"],
			wantTotal:  flashDef.ImagePrices["1K"],
			noUsage:    true,
		},
		{
			name: "pro session with 4K size",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: testProName,
					Size:  "4K",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "landscape"}}},
						{Role: "model", Parts: []*genai.Part{
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img1")}},
						}},
					},
				})
			},
			wantModel:  testProName,
			wantTurns:  1,
			wantImages: 1,
			wantImage:  proDef.ImagePrices["4K"],
			wantTotal:  proDef.ImagePrices["4K"],
			noUsage:    true,
		},
		{
			name: "legacy session no size defaults to 1K",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: testProName,
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hi"}}},
						{Role: "model", Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img")}}}},
					},
				})
			},
			wantModel:  testProName,
			wantTurns:  1,
			wantImages: 1,
			wantImage:  proDef.ImagePrices["1K"],
			wantTotal:  proDef.ImagePrices["1K"],
			noUsage:    true,
		},
		{
			name: "flash-3.1 session with 2K size",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: "flash-3.1",
					Size:  "2K",
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "a cat"}}},
						{Role: "model", Parts: []*genai.Part{
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img")}},
						}},
					},
				})
			},
			wantModel:  "flash-3.1",
			wantTurns:  1,
			wantImages: 1,
			wantImage:  flash31Def.ImagePrices["2K"],
			wantTotal:  flash31Def.ImagePrices["2K"],
			noUsage:    true,
		},
		{
			name: "usage data with zero images",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				data := fmt.Sprintf(`{"model":%q,"history":[{"role":"user","parts":[{"text":"describe this"}]},{"role":"model","parts":[{"text":"a landscape"}]}],"usage":{"prompt_tokens":500,"candidate_tokens":100,"total_tokens":600}}`, testFlashName)
				p := filepath.Join(dir, "test.session.json")
				os.WriteFile(p, []byte(data), 0644)
				return p
			},
			wantModel:  testFlashName,
			wantTurns:  1,
			wantImages: 0,
			wantInput:  float64(500) * flashDef.InputPerMTok / 1_000_000,
			wantOutput: float64(100) * flashDef.OutputPerMTok / 1_000_000,
			wantImage:  0,
			wantTotal:  float64(500)*flashDef.InputPerMTok/1_000_000 + float64(100)*flashDef.OutputPerMTok/1_000_000,
		},
		{
			name: "invalid path",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.session.json")
			},
			wantErr: "failed to read",
		},
		{
			name: "invalid JSON",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "bad.session.json")
				os.WriteFile(p, []byte("{not json"), 0644)
				return p
			},
			wantErr: "failed to parse",
		},
		{
			name: "missing history",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "no-hist.session.json")
				os.WriteFile(p, []byte(`{"model":"flash"}`), 0644)
				return p
			},
			wantErr: "not a banana session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			cb, err := analyzeSession(path)
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
			if cb.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", cb.Model, tt.wantModel)
			}
			if cb.Turns != tt.wantTurns {
				t.Errorf("Turns = %d, want %d", cb.Turns, tt.wantTurns)
			}
			if cb.OutputImages != tt.wantImages {
				t.Errorf("OutputImages = %d, want %d", cb.OutputImages, tt.wantImages)
			}
			if tt.noUsage && cb.Usage != nil {
				t.Errorf("Usage should be nil for legacy session")
			}
			assertClose := func(name string, got, want float64) {
				t.Helper()
				d := got - want
				if d < 0 {
					d = -d
				}
				if d > 0.000001 {
					t.Errorf("%s = %f, want %f", name, got, want)
				}
			}
			assertClose("InputCost", cb.InputCost, tt.wantInput)
			assertClose("OutputCost", cb.OutputCost, tt.wantOutput)
			assertClose("ImageCost", cb.ImageCost, tt.wantImage)
			assertClose("Total", cb.Total, tt.wantTotal)
		})
	}
}

func TestRunCost(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) []string
		wantErr string
	}{
		{
			name: "no args",
			setup: func(t *testing.T) []string {
				return []string{}
			},
			wantErr: "usage:",
		},
		{
			name: "single file",
			setup: func(t *testing.T) []string {
				dir := t.TempDir()
				p := writeSessionFile(t, dir, "test.session.json", sessionData{
					Model: testFlashName,
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hi"}}},
						{Role: "model", Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img")}}}},
					},
				})
				return []string{p}
			},
		},
		{
			name: "directory with mixed sessions",
			setup: func(t *testing.T) []string {
				dir := t.TempDir()
				writeSessionFile(t, dir, "a.session.json", sessionData{
					Model: testFlashName,
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hi"}}},
						{Role: "model", Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img")}}}},
					},
				})
				writeSessionFile(t, dir, "b.session.json", sessionData{
					Model: testProName,
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hi"}}},
						{Role: "model", Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("img")}}}},
					},
				})
				return []string{dir}
			},
		},
		{
			name: "directory ignores non-session files",
			setup: func(t *testing.T) []string {
				dir := t.TempDir()
				writeSessionFile(t, dir, "a.session.json", sessionData{
					Model: testFlashName,
					History: []*genai.Content{
						{Role: "user", Parts: []*genai.Part{{Text: "hi"}}},
					},
				})
				os.WriteFile(filepath.Join(dir, "notes.json"), []byte(`{}`), 0644)
				os.WriteFile(filepath.Join(dir, "image.png"), []byte("img"), 0644)
				return []string{dir}
			},
		},
		{
			name: "empty directory",
			setup: func(t *testing.T) []string {
				return []string{t.TempDir()}
			},
		},
		{
			name: "invalid path",
			setup: func(t *testing.T) []string {
				return []string{filepath.Join(t.TempDir(), "nonexistent")}
			},
			wantErr: "cannot access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.setup(t)
			err := runCost(args)
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

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		input int32
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1247, "1,247"},
		{12345, "12,345"},
		{1234567, "1,234,567"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.input), func(t *testing.T) {
			got := formatTokenCount(tt.input)
			if got != tt.want {
				t.Errorf("formatTokenCount(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.0, "0.0000"},
		{0.0003, "0.0003"},
		{0.009, "0.0090"},
		{0.01, "0.01"},
		{0.20, "0.20"},
		{1.50, "1.50"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatCost(tt.input)
			if got != tt.want {
				t.Errorf("formatCost(%f) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
