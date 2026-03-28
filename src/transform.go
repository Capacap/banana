package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

const transformUsage = `usage: agentpix transform -i <input> -o <output> [-f] <operation> [args]

operations:
  flip-h              horizontal flip (mirror)
  flip-v              vertical flip
  rotate <degrees>    rotate clockwise: 90, 180, 270
  resize <spec>       resize: WxH, Wx (proportional height), xH (proportional width)`

func runTransform(args []string) error {
	fs := flag.NewFlagSet("agentpix transform", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	input := fs.String("i", "", "input PNG file")
	output := fs.String("o", "", "output PNG file path")
	force := fs.Bool("f", false, "overwrite output if it exists")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%s", transformUsage)
	}

	if *input == "" || *output == "" {
		return fmt.Errorf("%s", transformUsage)
	}

	if strings.ToLower(filepath.Ext(*input)) != ".png" {
		return fmt.Errorf("input file %q is not a PNG (transform only supports .png)", *input)
	}
	if strings.ToLower(filepath.Ext(*output)) != ".png" {
		return fmt.Errorf("output file %q must be .png", *output)
	}

	if _, err := os.Stat(*input); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input file %q does not exist", *input)
		}
		return fmt.Errorf("cannot access input file %q: %v", *input, err)
	}
	if info, err := os.Stat(filepath.Dir(*output)); err != nil || !info.IsDir() {
		return fmt.Errorf("output directory %q does not exist", filepath.Dir(*output))
	}
	if _, err := os.Stat(*output); err == nil && !*force {
		return fmt.Errorf("output file %q already exists (use -f to overwrite)", *output)
	}

	opArgs := fs.Args()
	if len(opArgs) == 0 {
		return fmt.Errorf("%s", transformUsage)
	}

	inputData, err := os.ReadFile(*input)
	if err != nil {
		return fmt.Errorf("failed to read %q: %v", *input, err)
	}

	img, err := png.Decode(bytes.NewReader(inputData))
	if err != nil {
		return fmt.Errorf("failed to decode %q: %v", *input, err)
	}

	// Extract existing metadata to re-embed after transform.
	existingMeta, _ := pngGetText(inputData, metadataKey)

	var result image.Image
	switch opArgs[0] {
	case "flip-h":
		if len(opArgs) > 1 {
			return fmt.Errorf("flip-h takes no arguments, got: %s", strings.Join(opArgs[1:], " "))
		}
		result = flipH(img)
	case "flip-v":
		if len(opArgs) > 1 {
			return fmt.Errorf("flip-v takes no arguments, got: %s", strings.Join(opArgs[1:], " "))
		}
		result = flipV(img)
	case "rotate":
		if len(opArgs) < 2 {
			return fmt.Errorf("rotate requires a degrees argument: 90, 180, or 270")
		}
		if len(opArgs) > 2 {
			return fmt.Errorf("rotate takes one argument, got: %s", strings.Join(opArgs[1:], " "))
		}
		deg, err := strconv.Atoi(opArgs[1])
		if err != nil || (deg != 90 && deg != 180 && deg != 270) {
			return fmt.Errorf("invalid rotation %q: must be 90, 180, or 270", opArgs[1])
		}
		result = rotate(img, deg)
	case "resize":
		if len(opArgs) < 2 {
			return fmt.Errorf("resize requires a size spec: WxH, Wx, or xH")
		}
		if len(opArgs) > 2 {
			return fmt.Errorf("resize takes one argument, got: %s", strings.Join(opArgs[1:], " "))
		}
		result, err = resize(img, opArgs[1])
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown operation %q\n%s", opArgs[0], transformUsage)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, result); err != nil {
		return fmt.Errorf("failed to encode output PNG: %v", err)
	}

	outData := buf.Bytes()
	if existingMeta != "" {
		if tagged, err := pngSetText(outData, metadataKey, existingMeta); err == nil {
			outData = tagged
		}
	}

	if err := os.WriteFile(*output, outData, outputPerm); err != nil {
		return fmt.Errorf("failed to write %q: %v", *output, err)
	}
	fmt.Fprintf(os.Stderr, "saved %s (%d bytes)\n", *output, len(outData))
	return nil
}

func flipH(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.X-1-x+b.Min.X, y, img.At(x, y))
		}
	}
	return dst
}

func flipV(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, b.Max.Y-1-y+b.Min.Y, img.At(x, y))
		}
	}
	return dst
}

func rotate(img image.Image, degrees int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	switch degrees {
	case 90:
		dst := image.NewNRGBA(image.Rect(0, 0, h, w))
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				dst.Set(h-1-(y-b.Min.Y), x-b.Min.X, img.At(x, y))
			}
		}
		return dst
	case 180:
		dst := image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				dst.Set(w-1-(x-b.Min.X), h-1-(y-b.Min.Y), img.At(x, y))
			}
		}
		return dst
	case 270:
		dst := image.NewNRGBA(image.Rect(0, 0, h, w))
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				dst.Set(y-b.Min.Y, w-1-(x-b.Min.X), img.At(x, y))
			}
		}
		return dst
	default:
		return img
	}
}

func resize(img image.Image, spec string) (image.Image, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	targetW, targetH, err := parseResizeSpec(spec, w, h)
	if err != nil {
		return nil, err
	}

	dst := image.NewNRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst, nil
}

func parseResizeSpec(spec string, srcW, srcH int) (int, int, error) {
	parts := strings.SplitN(strings.ToLower(spec), "x", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid resize spec %q: use WxH, Wx, or xH", spec)
	}

	var targetW, targetH int
	var err error

	if parts[0] != "" {
		targetW, err = strconv.Atoi(parts[0])
		if err != nil || targetW <= 0 {
			return 0, 0, fmt.Errorf("invalid width in resize spec %q", spec)
		}
	}
	if parts[1] != "" {
		targetH, err = strconv.Atoi(parts[1])
		if err != nil || targetH <= 0 {
			return 0, 0, fmt.Errorf("invalid height in resize spec %q", spec)
		}
	}

	if targetW == 0 && targetH == 0 {
		return 0, 0, fmt.Errorf("resize spec %q must specify at least width or height", spec)
	}

	if targetW == 0 {
		targetW = srcW * targetH / srcH
		if targetW < 1 {
			targetW = 1
		}
	}
	if targetH == 0 {
		targetH = srcH * targetW / srcW
		if targetH < 1 {
			targetH = 1
		}
	}

	return targetW, targetH, nil
}
