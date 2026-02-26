package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/genai"
)

const metadataVersion = 1
const metadataKey = "banana"

type imageMetadata struct {
	Version   int           `json:"version"`
	Model     string        `json:"model"`
	ModelID   string        `json:"model_id"`
	Ratio     string        `json:"ratio"`
	Size      string        `json:"size,omitempty"`
	Inputs    []string      `json:"inputs,omitempty"`
	Session   string        `json:"session,omitempty"`
	Timestamp string        `json:"timestamp"`
	Prompts   []promptEntry `json:"prompts"`
}

type promptEntry struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

func buildMetadata(opts *options, history []*genai.Content) imageMetadata {
	var prompts []promptEntry
	for _, c := range history {
		if c == nil {
			continue
		}
		var textBuf strings.Builder
		for _, p := range c.Parts {
			if p == nil || p.InlineData != nil || p.Thought {
				continue
			}
			if p.Text != "" {
				if textBuf.Len() > 0 {
					textBuf.WriteByte('\n')
				}
				textBuf.WriteString(p.Text)
			}
		}
		if textBuf.Len() > 0 {
			prompts = append(prompts, promptEntry{Role: c.Role, Text: textBuf.String()})
		}
	}

	var inputs []string
	for _, p := range opts.inputs {
		inputs = append(inputs, filepath.Base(p))
	}

	var session string
	if opts.session != "" {
		session = filepath.Base(opts.session)
	}

	return imageMetadata{
		Version:   metadataVersion,
		Model:     opts.model,
		ModelID:   opts.modelID,
		Ratio:     opts.ratio,
		Size:      opts.size,
		Inputs:    inputs,
		Session:   session,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Prompts:   prompts,
	}
}

func embedMetadata(imageData []byte, meta imageMetadata) []byte {
	if !pngHasSignature(imageData) {
		fmt.Fprintln(os.Stderr, "note: output is not PNG, skipping metadata embedding")
		return imageData
	}
	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: failed to marshal metadata: %v\n", err)
		return imageData
	}
	result, err := pngSetText(imageData, metadataKey, string(jsonBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: failed to embed metadata: %v\n", err)
		return imageData
	}
	return result
}

func runMeta(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: banana meta <image.png>")
	}
	path := args[0]

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %q: %v", path, err)
	}

	if !pngHasSignature(data) {
		return fmt.Errorf("%q is not a PNG file (metadata is only embedded in PNG output)", path)
	}

	raw, err := pngGetText(data, metadataKey)
	if err != nil {
		return fmt.Errorf("no banana metadata found in %q", path)
	}

	var meta imageMetadata
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return fmt.Errorf("failed to parse metadata: %v", err)
	}

	fmt.Printf("version:   %d\n", meta.Version)
	fmt.Printf("model:     %s (%s)\n", meta.Model, meta.ModelID)
	fmt.Printf("ratio:     %s\n", meta.Ratio)
	if meta.Size != "" {
		fmt.Printf("size:      %s\n", meta.Size)
	}
	fmt.Printf("timestamp: %s\n", meta.Timestamp)
	if len(meta.Inputs) > 0 {
		fmt.Printf("inputs:    %s\n", strings.Join(meta.Inputs, ", "))
	}
	if meta.Session != "" {
		fmt.Printf("session:   %s\n", meta.Session)
	}

	if len(meta.Prompts) > 0 {
		fmt.Println()
		fmt.Println("prompts:")
		for i, p := range meta.Prompts {
			fmt.Printf("  [%d] %s: %s\n", i+1, p.Role, p.Text)
		}
	}

	return nil
}
