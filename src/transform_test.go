package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// testImage creates a w×h NRGBA image where each pixel color encodes its position.
func testImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 0, A: 255})
		}
	}
	return img
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write PNG: %v", err)
	}
}

func readPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %q: %v", path, err)
	}
	return img
}

func assertPixel(t *testing.T, img image.Image, x, y int, want color.NRGBA) {
	t.Helper()
	got := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	if got != want {
		t.Errorf("pixel(%d,%d) = %v, want %v", x, y, got, want)
	}
}

func TestFlipH(t *testing.T) {
	// 3x2 image, pixels encode position
	src := testImage(3, 2)
	dst := flipH(src)

	b := dst.Bounds()
	if b.Dx() != 3 || b.Dy() != 2 {
		t.Fatalf("bounds = %v, want 3x2", b)
	}

	// (0,0) should have what was at (2,0)
	assertPixel(t, dst, 0, 0, color.NRGBA{R: 2, G: 0, B: 0, A: 255})
	assertPixel(t, dst, 2, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	assertPixel(t, dst, 1, 1, color.NRGBA{R: 1, G: 1, B: 0, A: 255}) // center stays
}

func TestFlipV(t *testing.T) {
	src := testImage(2, 3)
	dst := flipV(src)

	b := dst.Bounds()
	if b.Dx() != 2 || b.Dy() != 3 {
		t.Fatalf("bounds = %v, want 2x3", b)
	}

	// (0,0) should have what was at (0,2)
	assertPixel(t, dst, 0, 0, color.NRGBA{R: 0, G: 2, B: 0, A: 255})
	assertPixel(t, dst, 0, 2, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
}

func TestRotate(t *testing.T) {
	// 3x2 image
	src := testImage(3, 2)

	t.Run("90", func(t *testing.T) {
		dst := rotate(src, 90)
		b := dst.Bounds()
		if b.Dx() != 2 || b.Dy() != 3 {
			t.Fatalf("bounds = %dx%d, want 2x3", b.Dx(), b.Dy())
		}
		// Top-left of rotated should be bottom-left of original: (0,1)
		assertPixel(t, dst, 0, 0, color.NRGBA{R: 0, G: 1, B: 0, A: 255})
	})

	t.Run("180", func(t *testing.T) {
		dst := rotate(src, 180)
		b := dst.Bounds()
		if b.Dx() != 3 || b.Dy() != 2 {
			t.Fatalf("bounds = %dx%d, want 3x2", b.Dx(), b.Dy())
		}
		// Top-left of rotated should be bottom-right of original: (2,1)
		assertPixel(t, dst, 0, 0, color.NRGBA{R: 2, G: 1, B: 0, A: 255})
	})

	t.Run("270", func(t *testing.T) {
		dst := rotate(src, 270)
		b := dst.Bounds()
		if b.Dx() != 2 || b.Dy() != 3 {
			t.Fatalf("bounds = %dx%d, want 2x3", b.Dx(), b.Dy())
		}
		// Top-left of rotated should be top-right of original: (2,0)
		assertPixel(t, dst, 0, 0, color.NRGBA{R: 2, G: 0, B: 0, A: 255})
	})
}

func TestResize(t *testing.T) {
	src := testImage(100, 50)

	t.Run("exact", func(t *testing.T) {
		dst, err := resize(src, "200x100")
		if err != nil {
			t.Fatal(err)
		}
		b := dst.Bounds()
		if b.Dx() != 200 || b.Dy() != 100 {
			t.Fatalf("bounds = %dx%d, want 200x100", b.Dx(), b.Dy())
		}
	})

	t.Run("width only", func(t *testing.T) {
		dst, err := resize(src, "200x")
		if err != nil {
			t.Fatal(err)
		}
		b := dst.Bounds()
		if b.Dx() != 200 || b.Dy() != 100 {
			t.Fatalf("bounds = %dx%d, want 200x100", b.Dx(), b.Dy())
		}
	})

	t.Run("height only", func(t *testing.T) {
		dst, err := resize(src, "x100")
		if err != nil {
			t.Fatal(err)
		}
		b := dst.Bounds()
		if b.Dx() != 200 || b.Dy() != 100 {
			t.Fatalf("bounds = %dx%d, want 200x100", b.Dx(), b.Dy())
		}
	})
}

func TestResizeProportionalClamp(t *testing.T) {
	// Extreme aspect ratio: 1x1000, resize to x1 would compute width = 0 without clamping.
	src := testImage(1, 200)
	dst, err := resize(src, "x1")
	if err != nil {
		t.Fatal(err)
	}
	b := dst.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 {
		t.Fatalf("degenerate output: %dx%d", b.Dx(), b.Dy())
	}
}

func TestParseResizeSpec(t *testing.T) {
	tests := []struct {
		spec    string
		wantErr bool
	}{
		{"100x200", false},
		{"100x", false},
		{"x200", false},
		{"x", true},    // no dimensions
		{"abc", true},   // no x separator
		{"0x100", true}, // zero width
		{"100x0", true}, // zero height
		{"-1x100", true},
	}

	for _, tt := range tests {
		_, _, err := parseResizeSpec(tt.spec, 100, 100)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseResizeSpec(%q): err=%v, wantErr=%v", tt.spec, err, tt.wantErr)
		}
	}
}

func TestRunTransformEndToEnd(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.png")
	output := filepath.Join(dir, "output.png")

	writePNG(t, input, testImage(4, 3))

	err := runTransform([]string{"-i", input, "-o", output, "flip-h"})
	if err != nil {
		t.Fatalf("runTransform: %v", err)
	}

	result := readPNG(t, output)
	b := result.Bounds()
	if b.Dx() != 4 || b.Dy() != 3 {
		t.Fatalf("output bounds = %dx%d, want 4x3", b.Dx(), b.Dy())
	}

	// Pixel (0,0) should be what was at (3,0) in original
	assertPixel(t, result, 0, 0, color.NRGBA{R: 3, G: 0, B: 0, A: 255})
}

func TestRunTransformPreservesMetadata(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.png")
	output := filepath.Join(dir, "output.png")

	// Create a PNG with embedded banana metadata.
	var buf bytes.Buffer
	png.Encode(&buf, testImage(4, 4))
	tagged, err := pngSetText(buf.Bytes(), metadataKey, `{"model":"flash"}`)
	if err != nil {
		t.Fatalf("pngSetText: %v", err)
	}
	os.WriteFile(input, tagged, 0644)

	if err := runTransform([]string{"-i", input, "-o", output, "flip-h"}); err != nil {
		t.Fatalf("runTransform: %v", err)
	}

	outData, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	val, err := pngGetText(outData, metadataKey)
	if err != nil {
		t.Fatalf("metadata lost after transform: %v", err)
	}
	if val != `{"model":"flash"}` {
		t.Errorf("metadata = %q, want %q", val, `{"model":"flash"}`)
	}
}

func TestRunTransformOverwriteProtection(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.png")
	output := filepath.Join(dir, "output.png")

	writePNG(t, input, testImage(2, 2))
	writePNG(t, output, testImage(2, 2)) // output already exists

	err := runTransform([]string{"-i", input, "-o", output, "flip-h"})
	if err == nil {
		t.Fatal("expected error for existing output without -f")
	}

	// With -f should succeed
	err = runTransform([]string{"-i", input, "-o", output, "-f", "flip-h"})
	if err != nil {
		t.Fatalf("runTransform with -f: %v", err)
	}
}

func TestRunTransformValidation(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.png")
	writePNG(t, input, testImage(2, 2))
	output := filepath.Join(dir, "output.png")

	tests := []struct {
		name string
		args []string
	}{
		{"missing input", []string{"-o", output, "flip-h"}},
		{"missing output", []string{"-i", input, "flip-h"}},
		{"missing operation", []string{"-i", input, "-o", output}},
		{"bad input ext", []string{"-i", "foo.jpg", "-o", output, "flip-h"}},
		{"bad output ext", []string{"-i", input, "-o", "foo.jpg", "flip-h"}},
		{"unknown op", []string{"-i", input, "-o", output, "spin"}},
		{"rotate no degrees", []string{"-i", input, "-o", output, "rotate"}},
		{"rotate bad degrees", []string{"-i", input, "-o", output, "rotate", "45"}},
		{"resize no spec", []string{"-i", input, "-o", output, "resize"}},
		{"trailing args flip-h", []string{"-i", input, "-o", output, "flip-h", "extra"}},
		{"trailing args rotate", []string{"-i", input, "-o", output, "rotate", "90", "extra"}},
		{"trailing args resize", []string{"-i", input, "-o", output, "resize", "100x100", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runTransform(tt.args); err == nil {
				t.Error("expected error")
			}
		})
	}
}
