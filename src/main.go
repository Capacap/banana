package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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

type options struct {
	prompt  string
	output  string
	inputs  stringSlice
	session string
	model   string // "flash" or "pro"
	modelID string // full model ID from models map
	ratio   string
	size    string // normalized: "" or "1K"/"2K"/"4K"
	force   bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := parseAndValidateFlags(args)
	if err != nil {
		return err
	}

	if err := validatePaths(opts); err != nil {
		return err
	}

	var history []*genai.Content
	if opts.session != "" {
		history, err = loadSession(opts.session, opts.model)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	config := &genai.GenerateContentConfig{
		ImageConfig: &genai.ImageConfig{
			AspectRatio: opts.ratio,
			ImageSize:   opts.size,
		},
	}

	chat, err := client.Chats.Create(ctx, opts.modelID, config, history)
	if err != nil {
		return fmt.Errorf("failed to create chat: %v", err)
	}

	// Build message parts
	var parts []genai.Part
	parts = append(parts, genai.Part{Text: opts.prompt})

	for _, path := range opts.inputs {
		imgData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read input image %q: %v", path, err)
		}
		mime, _ := mimeFromPath(path) // already validated
		parts = append(parts, genai.Part{InlineData: &genai.Blob{MIMEType: mime, Data: imgData}})
	}

	result, err := chat.SendMessage(ctx, parts...)
	if err != nil {
		return fmt.Errorf("generation failed: %v", err)
	}

	text, imageData, err := extractResult(result)
	if err != nil {
		return err
	}

	if text != "" {
		fmt.Println(text)
	}

	if err := os.WriteFile(opts.output, imageData, 0644); err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}
	fmt.Fprintf(os.Stderr, "saved %s (%d bytes)\n", opts.output, len(imageData))

	// Save session alongside output (never overwrite the source session)
	sessPath := sessionPath(opts.output)
	sessBytes, err := json.Marshal(sessionData{Model: opts.model, History: chat.History(true)})
	if err != nil {
		return fmt.Errorf("failed to serialize session: %v", err)
	}
	if err := os.WriteFile(sessPath, sessBytes, 0644); err != nil {
		return fmt.Errorf("failed to write session: %v", err)
	}
	fmt.Fprintf(os.Stderr, "session: %s\n", sessPath)

	return nil
}

func parseAndValidateFlags(args []string) (*options, error) {
	fs := flag.NewFlagSet("banana", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	prompt := fs.String("p", "", "text prompt (required)")
	output := fs.String("o", "", "output file path (required)")
	var inputs stringSlice
	fs.Var(&inputs, "i", "input image path (repeatable, for editing/reference)")
	session := fs.String("s", "", "session file to continue from")
	model := fs.String("m", "flash", "model: flash or pro")
	ratio := fs.String("r", "1:1", "aspect ratio: 1:1, 2:3, 3:2, 3:4, 4:3, 9:16, 16:9, 21:9")
	size := fs.String("z", "", "output size: 1k, 2k, or 4k (pro model only)")
	force := fs.Bool("f", false, "overwrite output and session files if they exist")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("usage: banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m flash|pro] [-r <ratio>] [-z 1k|2k|4k] [-f]")
	}

	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected arguments: %s\nusage: banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m flash|pro] [-r <ratio>] [-z 1k|2k|4k] [-f]", strings.Join(fs.Args(), " "))
	}

	if strings.TrimSpace(*prompt) == "" || *output == "" {
		return nil, fmt.Errorf("usage: banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m flash|pro] [-r <ratio>] [-z 1k|2k|4k] [-f]")
	}

	modelID, ok := models[*model]
	if !ok {
		return nil, fmt.Errorf("unknown model %q: use \"flash\" or \"pro\"", *model)
	}

	if !validRatios[*ratio] {
		return nil, fmt.Errorf("invalid aspect ratio %q", *ratio)
	}

	var imageSize string
	if *size != "" {
		normalized := strings.ToUpper(*size)
		if normalized != "1K" && normalized != "2K" && normalized != "4K" {
			return nil, fmt.Errorf("invalid size %q: use 1k, 2k, or 4k", *size)
		}
		if *model != "pro" {
			return nil, fmt.Errorf("-z (size) requires -m pro")
		}
		imageSize = normalized
	}

	if max := maxInputImages[*model]; len(inputs) > max {
		hint := ""
		if *model == "flash" {
			hint = "; use -m pro for up to 14"
		}
		return nil, fmt.Errorf("%s supports up to %d input images, got %d%s", *model, max, len(inputs), hint)
	}

	if os.Getenv("GOOGLE_API_KEY") == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is not set. Get one at https://aistudio.google.com")
	}

	if _, err := mimeFromPath(*output); err != nil {
		return nil, fmt.Errorf("output file %q has unsupported extension (supported: png, jpg/jpeg, webp, heic, heif)", *output)
	}

	return &options{
		prompt:  *prompt,
		output:  *output,
		inputs:  inputs,
		session: *session,
		model:   *model,
		modelID: modelID,
		ratio:   *ratio,
		size:    imageSize,
		force:   *force,
	}, nil
}

func validatePaths(opts *options) error {
	if info, err := os.Stat(filepath.Dir(opts.output)); err != nil || !info.IsDir() {
		return fmt.Errorf("output directory %q does not exist", filepath.Dir(opts.output))
	}

	if opts.session != "" && filepath.Clean(opts.output) == filepath.Clean(opts.session) {
		return fmt.Errorf("-o and -s must not point to the same file")
	}

	if _, err := os.Stat(opts.output); err == nil && !opts.force {
		return fmt.Errorf("output file %q already exists (use -f to overwrite)", opts.output)
	}

	if _, err := os.Stat(sessionPath(opts.output)); err == nil && !opts.force {
		return fmt.Errorf("session file %q already exists (use -f to overwrite)", sessionPath(opts.output))
	}

	for _, path := range opts.inputs {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("input file %q does not exist", path)
		} else if err != nil {
			return fmt.Errorf("cannot access input file %q: %v", path, err)
		}
		if _, err := mimeFromPath(path); err != nil {
			return fmt.Errorf("input file %q has unsupported extension (supported: png, jpg/jpeg, webp, heic, heif)", path)
		}
		if info.Size() > maxInputFileSize {
			return fmt.Errorf("input file %q is %.1f MB, exceeds 7 MB inline limit", path, float64(info.Size())/(1024*1024))
		}
	}

	return nil
}

func loadSession(path, model string) ([]*genai.Content, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session %q: %v", path, err)
	}
	var sess sessionData
	if err := json.Unmarshal(raw, &sess); err != nil {
		return nil, fmt.Errorf("failed to parse session %q: %v", path, err)
	}
	if sess.Model != "" && sess.Model != model {
		return nil, fmt.Errorf("session was created with %q but -m is %q; pass -m %s to continue this session", sess.Model, model, sess.Model)
	}
	return sess.History, nil
}

func extractResult(result *genai.GenerateContentResponse) (string, []byte, error) {
	if result == nil || len(result.Candidates) == 0 {
		if result != nil && result.PromptFeedback != nil && result.PromptFeedback.BlockReason != "" {
			return "", nil, fmt.Errorf("prompt blocked (reason: %s)", result.PromptFeedback.BlockReason)
		}
		debug, _ := json.MarshalIndent(result, "", "  ")
		return "", nil, fmt.Errorf("no response from model; raw response:\n%s", debug)
	}

	candidate := result.Candidates[0]
	if candidate.Content == nil {
		reason := "unknown"
		if candidate.FinishReason != "" {
			reason = string(candidate.FinishReason)
		}
		return "", nil, fmt.Errorf("generation blocked (reason: %s)", reason)
	}

	var textBuf strings.Builder
	var imageData []byte
	for _, part := range candidate.Content.Parts {
		if part == nil {
			continue
		}
		if part.Text != "" && !part.Thought {
			if textBuf.Len() > 0 {
				textBuf.WriteByte('\n')
			}
			textBuf.WriteString(part.Text)
		} else if part.InlineData != nil && len(part.InlineData.Data) > 0 && imageData == nil {
			imageData = part.InlineData.Data
		}
	}

	if imageData == nil {
		return "", nil, fmt.Errorf("model returned no image data")
	}

	return textBuf.String(), imageData, nil
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
