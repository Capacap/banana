package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type costBreakdown struct {
	File         string
	Model        string
	Size         string // resolved size for pricing: "1K", "2K", "4K"
	SizeFromData bool   // true if session contained explicit size data
	Turns        int
	OutputImages int
	Usage        *usageData
	InputCost    float64
	OutputCost   float64
	ImageCost    float64
	Total        float64
}

func analyzeSession(path string) (*costBreakdown, error) {
	sess, _, err := readSession(path)
	if err != nil {
		return nil, err
	}

	// Resolve model name (aliases to pinned names)
	model := sess.Model
	if pinned, ok := modelAliases[model]; ok {
		model = pinned
	}

	// Count output images from model-role parts
	var outputImages int
	for _, c := range sess.History {
		if c == nil || c.Role != "model" {
			continue
		}
		for _, p := range c.Parts {
			if p != nil && p.InlineData != nil && len(p.InlineData.Data) > 0 {
				outputImages++
			}
		}
	}

	turns := (len(sess.History) + 1) / 2

	size := sess.Size
	sizeFromData := size != ""
	if size == "" {
		size = "1K"
	}

	cb := &costBreakdown{
		File:         filepath.Base(path),
		Model:        model,
		Size:         size,
		SizeFromData: sizeFromData,
		Turns:        turns,
		OutputImages: outputImages,
		Usage:        sess.Usage,
	}

	def, known := modelDefs[model]
	if !known {
		return cb, nil
	}

	if sess.Usage != nil {
		cb.InputCost = float64(sess.Usage.PromptTokens) * def.InputPerMTok / 1_000_000
		cb.OutputCost = float64(sess.Usage.CandidateTokens) * def.OutputPerMTok / 1_000_000
	}
	if price, ok := def.ImagePrices[size]; ok {
		cb.ImageCost = float64(outputImages) * price
	} else {
		cb.ImageCost = float64(outputImages) * def.ImagePrices["1K"]
	}
	cb.Total = cb.InputCost + cb.OutputCost + cb.ImageCost

	return cb, nil
}

func runCost(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: banana cost <session-file-or-directory>")
	}
	target := args[0]

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("cannot access %q: %v", target, err)
	}

	if !info.IsDir() {
		return runCostFile(target)
	}
	return runCostDir(target)
}

func runCostFile(path string) error {
	cb, err := analyzeSession(path)
	if err != nil {
		return err
	}

	_, known := modelDefs[cb.Model]

	fmt.Printf("model:   %s\n", cb.Model)
	fmt.Printf("turns:   %d\n", cb.Turns)

	if cb.Usage != nil && known {
		fmt.Printf("input:   %s tokens ($%s)\n", formatTokenCount(cb.Usage.PromptTokens), formatCost(cb.InputCost))
		fmt.Printf("output:  %s tokens ($%s)\n", formatTokenCount(cb.Usage.CandidateTokens), formatCost(cb.OutputCost))
	} else if cb.Usage != nil {
		fmt.Printf("input:   %s tokens\n", formatTokenCount(cb.Usage.PromptTokens))
		fmt.Printf("output:  %s tokens\n", formatTokenCount(cb.Usage.CandidateTokens))
	} else {
		fmt.Printf("tokens:  no data\n")
	}

	if known {
		sizeNote := cb.Size
		if !cb.SizeFromData {
			sizeNote += " (assumed)"
		}
		fmt.Printf("images:  %d @ %s ($%s)\n", cb.OutputImages, sizeNote, formatCost(cb.ImageCost))
		fmt.Printf("total:   ~$%s\n", formatCost(cb.Total))
	} else {
		fmt.Printf("images:  %d\n", cb.OutputImages)
		fmt.Printf("total:   unknown (unrecognized model)\n")
	}

	fmt.Printf("\nprices collected %s\n", pricesCollected)
	return nil
}

func runCostDir(dir string) error {
	paths, err := listSessionFiles(dir)
	if err != nil {
		return err
	}

	var results []*costBreakdown
	for _, path := range paths {
		cb, err := analyzeSession(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", filepath.Base(path), err)
			continue
		}
		results = append(results, cb)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "no session files found")
		return nil
	}

	var totalCost float64
	var totalImages int
	var unpriced int
	for _, cb := range results {
		_, known := modelDefs[cb.Model]
		costStr := "?"
		if known {
			costStr = fmt.Sprintf("~$%s", formatCost(cb.Total))
		} else {
			unpriced++
		}
		sizeStr := cb.Size
		if !cb.SizeFromData {
			sizeStr += "?"
		}
		fmt.Printf("  %-30s %-10s %-3s turns=%-3d images=%-3d %s\n", cb.File, cb.Model, sizeStr, cb.Turns, cb.OutputImages, costStr)
		totalCost += cb.Total
		totalImages += cb.OutputImages
	}

	totalLine := fmt.Sprintf("\n  total: %d sessions, %d images, ~$%s", len(results), totalImages, formatCost(totalCost))
	if unpriced > 0 {
		totalLine += fmt.Sprintf(" (%d unpriced)", unpriced)
	}
	fmt.Println(totalLine)
	fmt.Printf("\nprices collected %s\n", pricesCollected)
	return nil
}

func formatCost(usd float64) string {
	if usd < 0.01 {
		return fmt.Sprintf("%.4f", usd)
	}
	return fmt.Sprintf("%.2f", usd)
}

func formatTokenCount(n int32) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var buf strings.Builder
	offset := len(s) % 3
	if offset == 0 {
		offset = 3
	}
	buf.WriteString(s[:offset])
	for i := offset; i < len(s); i += 3 {
		buf.WriteByte(',')
		buf.WriteString(s[i : i+3])
	}
	return buf.String()
}
