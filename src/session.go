package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/genai"
)

const sessionSuffix = ".session.json"

type usageData struct {
	PromptTokens    int32 `json:"prompt_tokens"`
	CandidateTokens int32 `json:"candidate_tokens"`
	TotalTokens     int32 `json:"total_tokens"`
}

type sessionData struct {
	Model   string           `json:"model"`
	Size    string           `json:"size,omitempty"`
	History []*genai.Content `json:"history"`
	Usage   *usageData       `json:"usage,omitempty"`
}

// readSession parses a session file and returns the session data and file size.
// It validates that history is present but does not check model names.
func readSession(path string) (*sessionData, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read %q: %v", path, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read %q: %v", path, err)
	}
	var sess sessionData
	if err := json.Unmarshal(raw, &sess); err != nil {
		return nil, 0, fmt.Errorf("failed to parse %q: %v", path, err)
	}
	if sess.History == nil {
		return nil, 0, fmt.Errorf("%q is not a banana session", path)
	}
	return &sess, info.Size(), nil
}

// listSessionFiles returns paths to all .session.json files in a directory (non-recursive).
func listSessionFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory: %v", err)
	}
	var paths []string
	for _, d := range entries {
		if d.IsDir() || !strings.HasSuffix(d.Name(), sessionSuffix) {
			continue
		}
		paths = append(paths, filepath.Join(dir, d.Name()))
	}
	return paths, nil
}

// sessionPath converts an output image path to the corresponding session file path.
func sessionPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	return strings.TrimSuffix(outputPath, ext) + sessionSuffix
}

// cleanHistoryForResume prepares saved history for use as chat context.
// Flash 3.1+ requires thought signatures on every part in model turns when
// thinking was active. The SDK only attaches signatures to image output parts,
// not the accompanying text. Sending text parts without signatures causes a
// 400 error. Since model text is just commentary ("Here is your image..."),
// we drop unsigned parts from model turns. User turns are passed through.
func cleanHistoryForResume(history []*genai.Content) []*genai.Content {
	for _, content := range history {
		if content.Role != "model" {
			continue
		}
		// Check whether any part in this turn carries a thought signature.
		hasSignedPart := false
		for _, part := range content.Parts {
			if part != nil && part.ThoughtSignature != nil {
				hasSignedPart = true
				break
			}
		}
		if !hasSignedPart {
			continue // no signing in this turn; leave it alone
		}
		// Keep only parts that carry a thought signature.
		var signed []*genai.Part
		for _, part := range content.Parts {
			if part != nil && part.ThoughtSignature != nil {
				signed = append(signed, part)
			}
		}
		content.Parts = signed
	}
	return history
}

// loadSession reads a session file for continuation, validating that its model
// matches the requested model. Returns the conversation history.
func loadSession(path, model string) ([]*genai.Content, error) {
	sess, _, err := readSession(path)
	if err != nil {
		return nil, err
	}
	if sess.Model != "" && sess.Model != model {
		// Legacy sessions stored bare aliases ("flash", "pro"); allow if same family
		if target, isAlias := modelAliases[sess.Model]; isAlias && modelDefs[target].Family == modelDefs[model].Family {
			return cleanHistoryForResume(sess.History), nil
		}
		return nil, fmt.Errorf("session was created with %q but -m is %q; pass -m %s to continue this session", sess.Model, model, sess.Model)
	}
	return cleanHistoryForResume(sess.History), nil
}
