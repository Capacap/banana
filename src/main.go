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
	"sort"
	"strings"

	"google.golang.org/genai"
)

type modelDef struct {
	ID             string
	Family         string
	MaxInputImages int
	InputPerMTok   float64            // USD per 1M input tokens
	OutputPerMTok  float64            // USD per 1M output text tokens
	Sizes          []string           // supported output sizes, e.g. ["1K"] or ["1K","2K","4K"]
	ImagePrices    map[string]float64 // size -> per-image USD cost
}

const pricesCollected = "2026-02-26"

var modelDefs = map[string]modelDef{
	"flash-2.5": {ID: "gemini-2.5-flash-image", Family: "flash", MaxInputImages: 3, InputPerMTok: 0.30, OutputPerMTok: 0.60, Sizes: []string{"1K"}, ImagePrices: map[string]float64{"1K": 0.039}},
	"flash-3.1": {ID: "gemini-3.1-flash-image-preview", Family: "flash", MaxInputImages: 14, InputPerMTok: 0.25, OutputPerMTok: 1.50, Sizes: []string{"1K", "2K", "4K"}, ImagePrices: map[string]float64{"1K": 0.067, "2K": 0.101, "4K": 0.151}},
	"pro-3.0":   {ID: "gemini-3-pro-image-preview", Family: "pro", MaxInputImages: 14, InputPerMTok: 2.00, OutputPerMTok: 12.00, Sizes: []string{"1K", "2K", "4K"}, ImagePrices: map[string]float64{"1K": 0.134, "2K": 0.134, "4K": 0.240}},
}

var modelAliases = map[string]string{
	"flash": "flash-3.1",
	"pro":   "pro-3.0",
}

var validRatios = map[string]bool{
	"1:1": true, "2:3": true, "3:2": true, "3:4": true, "4:3": true,
	"9:16": true, "16:9": true, "21:9": true,
}

func isKnownModel(name string) bool {
	if _, ok := modelDefs[name]; ok {
		return true
	}
	_, ok := modelAliases[name]
	return ok
}

func validModelNames() string {
	seen := make(map[string]bool)
	var names []string
	for name := range modelAliases {
		seen[name] = true
		names = append(names, name)
	}
	for name := range modelDefs {
		if !seen[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

const maxInputFileSize = 7 * 1024 * 1024 // 7 MB inline limit
const outputPerm = 0644

type stringSlice []string

func (s *stringSlice) String() string    { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

type options struct {
	prompt  string
	output  string
	inputs  stringSlice
	session string
	model   string // resolved name: "flash-3.1", "flash-2.5", "pro-3.0"
	modelID string // full model ID from modelDefs map
	ratio   string
	size    string // normalized: "" or "1K"/"2K"/"4K"
	force   bool
}

const usageText = `usage: banana -p <prompt> -o <output> [flags]
       banana meta <image.png>
       banana cost <session-file-or-directory>
       banana clean [-f] <directory>

flags:
  -p   text prompt (required)
  -o   output PNG file path (required)
  -i   input image, repeatable (flash-2.5: 3 max, others: 14 max)
  -s   session file to continue from
  -m   model: flash (default), pro, flash-2.5, flash-3.1, pro-3.0
  -r   aspect ratio: 1:1 (default), 2:3, 3:2, 3:4, 4:3, 9:16, 16:9, 21:9
  -z   output size: 1K, 2K, 4K (flash-3.1, pro-3.0)
  -f   overwrite existing output and session files

subcommands:
  meta    show metadata embedded in a generated PNG
  cost    estimate API cost from session files
  clean   find and remove session files from a directory`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", usageText)
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Println(usageText)
		return nil
	case "clean":
		return runClean(args[1:])
	case "cost":
		return runCost(args[1:])
	case "meta":
		return runMeta(args[1:])
	}

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

	meta := buildMetadata(opts, chat.History(true))
	imageData = embedMetadata(imageData, meta)

	if err := os.WriteFile(opts.output, imageData, outputPerm); err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}
	fmt.Fprintf(os.Stderr, "saved %s (%d bytes)\n", opts.output, len(imageData))

	// Save session alongside output (never overwrite the source session).
	// Usage reflects this API call only; multi-turn sessions accumulate input
	// tokens across calls, but we only record the final call's counts.
	sessPath := sessionPath(opts.output)
	var usage *usageData
	if result.UsageMetadata != nil {
		usage = &usageData{
			PromptTokens:    result.UsageMetadata.PromptTokenCount,
			CandidateTokens: result.UsageMetadata.CandidatesTokenCount,
			TotalTokens:     result.UsageMetadata.TotalTokenCount,
		}
	}
	sessBytes, err := json.Marshal(sessionData{Model: opts.model, Size: opts.size, History: chat.History(true), Usage: usage})
	if err != nil {
		return fmt.Errorf("failed to serialize session: %v", err)
	}
	if err := os.WriteFile(sessPath, sessBytes, outputPerm); err != nil {
		return fmt.Errorf("failed to write session: %v", err)
	}
	fmt.Fprintf(os.Stderr, "session: %s\n", sessPath)

	return nil
}

func parseAndValidateFlags(args []string) (*options, error) {
	fs := flag.NewFlagSet("banana", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	prompt := fs.String("p", "", "text prompt (required)")
	output := fs.String("o", "", "output PNG file path (required)")
	var inputs stringSlice
	fs.Var(&inputs, "i", "input image path (repeatable, for editing/reference)")
	session := fs.String("s", "", "session file to continue from")
	model := fs.String("m", "flash", "model name")
	ratio := fs.String("r", "1:1", "aspect ratio: 1:1, 2:3, 3:2, 3:4, 4:3, 9:16, 16:9, 21:9")
	size := fs.String("z", "", "output size: 1K, 2K, or 4K (flash-3.1, pro-3.0)")
	force := fs.Bool("f", false, "overwrite output and session files if they exist")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("usage: banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m model] [-r <ratio>] [-z 1K|2K|4K] [-f]")
	}

	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected arguments: %s\nusage: banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m model] [-r <ratio>] [-z 1K|2K|4K] [-f]", strings.Join(fs.Args(), " "))
	}

	if strings.TrimSpace(*prompt) == "" || *output == "" {
		return nil, fmt.Errorf("usage: banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m model] [-r <ratio>] [-z 1K|2K|4K] [-f]")
	}

	resolved := *model
	if pinned, ok := modelAliases[resolved]; ok {
		resolved = pinned
	}

	def, ok := modelDefs[resolved]
	if !ok {
		return nil, fmt.Errorf("unknown model %q: valid models are %s", *model, validModelNames())
	}

	if !validRatios[*ratio] {
		return nil, fmt.Errorf("invalid aspect ratio %q", *ratio)
	}

	var imageSize string
	if *size != "" {
		normalized := strings.ToUpper(*size)
		if len(def.Sizes) <= 1 {
			var alts []string
			for name, d := range modelDefs {
				if len(d.Sizes) > 1 {
					alts = append(alts, name)
				}
			}
			sort.Strings(alts)
			return nil, fmt.Errorf("-z is not supported by %s; models with size control: %s", resolved, strings.Join(alts, ", "))
		}
		found := false
		for _, s := range def.Sizes {
			if s == normalized {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("invalid size %q for %s: supported sizes are %s", normalized, resolved, strings.Join(def.Sizes, ", "))
		}
		imageSize = normalized
	}

	if len(inputs) > def.MaxInputImages {
		hint := ""
		var maxAvail int
		for _, d := range modelDefs {
			if d.MaxInputImages > maxAvail {
				maxAvail = d.MaxInputImages
			}
		}
		if def.MaxInputImages < maxAvail {
			var alts []string
			for name, d := range modelDefs {
				if d.MaxInputImages > def.MaxInputImages {
					alts = append(alts, name)
				}
			}
			sort.Strings(alts)
			hint = fmt.Sprintf("; %s support up to %d", strings.Join(alts, ", "), maxAvail)
		}
		return nil, fmt.Errorf("%s supports up to %d input images, got %d%s", resolved, def.MaxInputImages, len(inputs), hint)
	}

	if os.Getenv("GOOGLE_API_KEY") == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is not set. Get one at https://aistudio.google.com")
	}

	if strings.ToLower(filepath.Ext(*output)) != ".png" {
		return nil, fmt.Errorf("output file must be .png (the Gemini API returns PNG data)")
	}

	return &options{
		prompt:  *prompt,
		output:  *output,
		inputs:  inputs,
		session: *session,
		model:   resolved,
		modelID: def.ID,
		ratio:   *ratio,
		size:    imageSize,
		force:   *force,
	}, nil
}

func validatePaths(opts *options) error {
	if info, err := os.Stat(filepath.Dir(opts.output)); err != nil || !info.IsDir() {
		return fmt.Errorf("output directory %q does not exist", filepath.Dir(opts.output))
	}

	sessOut := sessionPath(opts.output)

	if opts.session != "" && !opts.force && filepath.Clean(sessOut) == filepath.Clean(opts.session) {
		return fmt.Errorf("session save path %q collides with -s source (use -f to overwrite)", sessOut)
	}

	if _, err := os.Stat(opts.output); err == nil && !opts.force {
		return fmt.Errorf("output file %q already exists (use -f to overwrite)", opts.output)
	}

	if _, err := os.Stat(sessOut); err == nil && !opts.force {
		return fmt.Errorf("session file %q already exists (use -f to overwrite)", sessOut)
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
			return fmt.Errorf("input file %q is %.1f MB, exceeds %.0f MB inline limit", path, float64(info.Size())/(1024*1024), float64(maxInputFileSize)/(1024*1024))
		}
	}

	return nil
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
