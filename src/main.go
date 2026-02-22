package main

import (
	"context"
	"encoding/json"
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
	"4:5": true, "5:4": true, "9:16": true, "16:9": true, "21:9": true,
}

func main() {
	prompt := flag.String("p", "", "text prompt (required)")
	output := flag.String("o", "", "output file path (required)")
	input := flag.String("i", "", "input image path (for editing)")
	session := flag.String("s", "", "session file to continue from")
	model := flag.String("m", "flash", "model: flash or pro")
	ratio := flag.String("r", "1:1", "aspect ratio: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9")
	flag.Parse()

	if *prompt == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: banana -p <prompt> -o <output> [-i <input>] [-s <session>] [-m flash|pro] [-r <ratio>]")
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

	// Load session history if continuing
	var history []*genai.Content
	if *session != "" {
		raw, err := os.ReadFile(*session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read session %q: %v\n", *session, err)
			os.Exit(1)
		}
		if err := json.Unmarshal(raw, &history); err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse session %q: %v\n", *session, err)
			os.Exit(1)
		}
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
		},
	}

	chat, err := client.Chats.Create(ctx, modelID, config, history)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create chat: %v\n", err)
		os.Exit(1)
	}

	// Build message parts
	var parts []genai.Part
	parts = append(parts, *genai.NewPartFromText(*prompt))

	if *input != "" {
		imgData, err := os.ReadFile(*input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read input image: %v\n", err)
			os.Exit(1)
		}
		mime, err := mimeFromPath(*input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		parts = append(parts, genai.Part{InlineData: &genai.Blob{MIMEType: mime, Data: imgData}})
	}

	result, err := chat.SendMessage(ctx, parts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generation failed: %v\n", err)
		os.Exit(1)
	}

	if result == nil || len(result.Candidates) == 0 {
		fmt.Fprintln(os.Stderr, "no response from model")
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
		if part.Text != "" {
			fmt.Println(part.Text)
		} else if part.InlineData != nil {
			if err := os.WriteFile(*output, part.InlineData.Data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("saved %s (%d bytes)\n", *output, len(part.InlineData.Data))
			saved = true
		}
	}

	if !saved {
		fmt.Fprintln(os.Stderr, "model returned no image data")
		os.Exit(1)
	}

	// Save session
	sessPath := sessionPath(*output)
	historyData, err := json.Marshal(chat.History(false))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to serialize session: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(sessPath, historyData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write session: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("session: %s\n", sessPath)
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
		return "", fmt.Errorf("unsupported image format %q (supported: png, jpg, webp, heic, heif)", ext)
	}
	return mime, nil
}
