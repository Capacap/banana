package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type sessionInfo struct {
	Model string
	Turns int
	Size  int64
}

func validateSessionFile(path string) (*sessionInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot stat file: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %v", err)
	}

	var sess sessionData
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&sess); err != nil {
		return nil, fmt.Errorf("not a banana session: %v", err)
	}

	if sess.Model != "" {
		if _, ok := models[sess.Model]; !ok {
			return nil, fmt.Errorf("unknown model %q", sess.Model)
		}
	}

	if sess.History == nil {
		return nil, fmt.Errorf("missing history field")
	}

	return &sessionInfo{
		Model: sess.Model,
		Turns: (len(sess.History) + 1) / 2,
		Size:  info.Size(),
	}, nil
}

func runClean(args []string) error {
	fs := flag.NewFlagSet("banana clean", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	force := fs.Bool("f", false, "delete validated session files (without -f, dry-run only)")

	const usage = "find session files and report sizes (add -f to delete)\nusage: banana clean [-f] <directory>"

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf(usage)
	}

	if fs.NArg() != 1 {
		if fs.NArg() > 1 {
			for _, a := range fs.Args()[1:] {
				if a == "-f" {
					return fmt.Errorf("flag -f must appear before the directory\n" + usage)
				}
			}
		}
		return fmt.Errorf(usage)
	}
	dir := fs.Arg(0)

	stat, err := os.Stat(dir)
	if err != nil || !stat.IsDir() {
		return fmt.Errorf("%q is not a directory", dir)
	}

	type validatedFile struct {
		path string
		info *sessionInfo
	}

	var files []validatedFile
	var skipped int

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("cannot read directory: %v", err)
	}
	for _, d := range entries {
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".session.json") {
			continue
		}
		path := filepath.Join(dir, d.Name())
		si, vErr := validateSessionFile(path)
		if vErr != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", path, vErr)
			skipped++
			continue
		}
		files = append(files, validatedFile{path: path, info: si})
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no session files found")
		return nil
	}

	var totalSize int64
	for _, f := range files {
		model := f.info.Model
		if model == "" {
			model = "legacy"
		}
		fmt.Printf("  %s  model=%s turns=%d size=%s\n", f.path, model, f.info.Turns, formatSize(f.info.Size))
		totalSize += f.info.Size
	}

	if !*force {
		fmt.Printf("\ndry run: %d files, %s would be freed", len(files), formatSize(totalSize))
		if skipped > 0 {
			fmt.Printf(" (%d skipped)", skipped)
		}
		fmt.Println()
		return nil
	}

	var deleted int
	var freed int64
	for _, f := range files {
		if err := os.Remove(f.path); err != nil {
			fmt.Fprintf(os.Stderr, "failed to delete %s: %v\n", f.path, err)
			continue
		}
		deleted++
		freed += f.info.Size
	}

	fmt.Printf("deleted %d files, freed %s", deleted, formatSize(freed))
	if skipped > 0 {
		fmt.Printf(" (%d skipped)", skipped)
	}
	fmt.Println()

	return nil
}

func formatSize(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
