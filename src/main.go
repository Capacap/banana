package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/genai"
)

var models = map[string]string{
	"flash": "gemini-2.5-flash-image",
	"pro":   "gemini-3-pro-image-preview",
}

var validRatios = map[string]bool{
	"1:1": true, "2:3": true, "3:2": true, "3:4": true, "4:3": true,
	"9:16": true, "16:9": true, "21:9": true,
}

var maxInputImages = map[string]int{"flash": 3, "pro": 14}

const maxInputFileSize = 7 * 1024 * 1024 // 7 MB inline limit

type sessionData struct {
	Model   string           `json:"model"`
	History []*genai.Content `json:"history"`
}

type stringSlice []string

func (s *stringSlice) String() string    { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	prompt := flag.String("p", "", "text prompt (required)")
	output := flag.String("o", "", "output file path (required)")
	var inputs stringSlice
	flag.Var(&inputs, "i", "input image path (repeatable, for editing/reference)")
	session := flag.String("s", "", "session file to continue from")
	model := flag.String("m", "flash", "model: flash or pro")
	ratio := flag.String("r", "1:1", "aspect ratio: 1:1, 2:3, 3:2, 3:4, 4:3, 9:16, 16:9, 21:9")
	size := flag.String("z", "", "output size: 1k, 2k, or 4k (pro model only)")
	force := flag.Bool("f", false, "overwrite output and session files if they exist")
	flag.Parse()

	if strings.TrimSpace(*prompt) == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m flash|pro] [-r <ratio>] [-z 1k|2k|4k] [-f]")
		os.Exit(1)
	}

	modelID, ok := models[*model]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown model %q: use \"flash\" or \"pro\"\n", *model)
		os.Exit(1)
	}

	if !validRatios[*ratio] {
		fmt.Fprintf(os.Stderr, "invalid aspect ratio %q\n", *ratio)
		os.Exit(1)
	}

	var imageSize string
	if *size != "" {
		normalized := strings.ToUpper(*size)
		if normalized != "1K" && normalized != "2K" && normalized != "4K" {
			fmt.Fprintf(os.Stderr, "invalid size %q: use 1k, 2k, or 4k\n", *size)
			os.Exit(1)
		}
		if *model != "pro" {
			fmt.Fprintln(os.Stderr, "-z (size) requires -m pro")
			os.Exit(1)
		}
		imageSize = normalized
	}

	if max := maxInputImages[*model]; len(inputs) > max {
		hint := ""
		if *model == "flash" {
			hint = "; use -m pro for up to 14"
		}
		fmt.Fprintf(os.Stderr, "%s supports up to %d input images, got %d%s\n", *model, max, len(inputs), hint)
		os.Exit(1)
	}

	if os.Getenv("GOOGLE_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "GOOGLE_API_KEY is not set. Get one at https://aistudio.google.com")
		os.Exit(1)
	}

	if _, err := mimeFromPath(*output); err != nil {
		fmt.Fprintf(os.Stderr, "output file %q has unsupported extension (supported: png, jpg/jpeg, webp, heic, heif)\n", *output)
		os.Exit(1)
	}

	if info, err := os.Stat(filepath.Dir(*output)); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "output directory %q does not exist\n", filepath.Dir(*output))
		os.Exit(1)
	}

	if *session != "" && filepath.Clean(*output) == filepath.Clean(*session) {
		fmt.Fprintln(os.Stderr, "-o and -s must not point to the same file")
		os.Exit(1)
	}

	if _, err := os.Stat(*output); err == nil && !*force {
		fmt.Fprintf(os.Stderr, "output file %q already exists (use -f to overwrite)\n", *output)
		os.Exit(1)
	}

	if *session == "" {
		if _, err := os.Stat(sessionPath(*output)); err == nil && !*force {
			fmt.Fprintf(os.Stderr, "session file %q already exists (use -f to overwrite)\n", sessionPath(*output))
			os.Exit(1)
		}
	}

	for _, path := range inputs {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "input file %q does not exist\n", path)
			os.Exit(1)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "cannot access input file %q: %v\n", path, err)
			os.Exit(1)
		}
		if _, err := mimeFromPath(path); err != nil {
			fmt.Fprintf(os.Stderr, "input file %q has unsupported extension (supported: png, jpg/jpeg, webp, heic, heif)\n", path)
			os.Exit(1)
		}
		if info.Size() > maxInputFileSize {
			fmt.Fprintf(os.Stderr, "input file %q is %.1f MB, exceeds 7 MB inline limit\n", path, float64(info.Size())/(1024*1024))
			os.Exit(1)
		}
	}

	// Load session history if continuing
	var history []*genai.Content
	if *session != "" {
		raw, err := os.ReadFile(*session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read session %q: %v\n", *session, err)
			os.Exit(1)
		}
		var sess sessionData
		if err := json.Unmarshal(raw, &sess); err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse session %q: %v\n", *session, err)
			os.Exit(1)
		}
		if sess.Model != "" && sess.Model != *model {
			fmt.Fprintf(os.Stderr, "session was created with %q but -m is %q; pass -m %s to continue this session\n", sess.Model, *model, sess.Model)
			os.Exit(1)
		}
		history = sess.History
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create client: %v\n", err)
		os.Exit(1)
	}

	config := &genai.GenerateContentConfig{
		ImageConfig: &genai.ImageConfig{
			AspectRatio: *ratio,
			ImageSize:   imageSize,
		},
	}

	chat, err := client.Chats.Create(ctx, modelID, config, history)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create chat: %v\n", err)
		os.Exit(1)
	}

	// Build message parts
	var parts []genai.Part
	parts = append(parts, genai.Part{Text: *prompt})

	for _, path := range inputs {
		imgData, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read input image %q: %v\n", path, err)
			os.Exit(1)
		}
		mime, _ := mimeFromPath(path) // already validated
		parts = append(parts, genai.Part{InlineData: &genai.Blob{MIMEType: mime, Data: imgData}})
	}

	result, err := chat.SendMessage(ctx, parts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generation failed: %v\n", err)
		os.Exit(1)
	}

	if result == nil || len(result.Candidates) == 0 {
		if result != nil && result.PromptFeedback != nil && result.PromptFeedback.BlockReason != "" {
			fmt.Fprintf(os.Stderr, "prompt blocked (reason: %s)\n", result.PromptFeedback.BlockReason)
		} else {
			debug, _ := json.MarshalIndent(result, "", "  ")
			fmt.Fprintf(os.Stderr, "no response from model; raw response:\n%s\n", debug)
		}
		os.Exit(1)
	}

	candidate := result.Candidates[0]
	if candidate.Content == nil {
		reason := "unknown"
		if candidate.FinishReason != "" {
			reason = string(candidate.FinishReason)
		}
		fmt.Fprintf(os.Stderr, "generation blocked (reason: %s)\n", reason)
		os.Exit(1)
	}

	saved := false
	for _, part := range candidate.Content.Parts {
		if part == nil {
			continue
		}
		if part.Text != "" && !part.Thought {
			fmt.Println(part.Text)
		} else if part.InlineData != nil && len(part.InlineData.Data) > 0 && !saved {
			if err := os.WriteFile(*output, part.InlineData.Data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "saved %s (%d bytes)\n", *output, len(part.InlineData.Data))
			saved = true
		}
	}

	if !saved {
		fmt.Fprintln(os.Stderr, "model returned no image data")
		os.Exit(1)
	}

	// Save session
	sessPath := sessionPath(*output)
	if *session != "" {
		sessPath = *session
	}
	sessData, err := json.Marshal(sessionData{Model: *model, History: chat.History(true)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to serialize session: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(sessPath, sessData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write session: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "session: %s\n", sessPath)
}

func sessionPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	return strings.TrimSuffix(outputPath, ext) + ".session.json"
}

var supportedMimes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".heic": "image/heic",
	".heif": "image/heif",
}

func mimeFromPath(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	mime, ok := supportedMimes[ext]
	if !ok {
		return "", fmt.Errorf("unsupported image format %q (supported: png, jpg/jpeg, webp, heic, heif)", ext)
	}
	return mime, nil
}
