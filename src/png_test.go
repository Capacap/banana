package main

import (
	"encoding/binary"
	"hash/crc32"
	"strings"
	"testing"
)

// minimalPNG builds the smallest valid PNG: signature + IHDR + IDAT + IEND.
// The image is 1x1 pixel, 8-bit grayscale, with a single zero-filtered row.
func minimalPNG() []byte {
	var buf []byte
	// Signature
	buf = append(buf, 0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A)
	// IHDR: width=1, height=1, bit_depth=8, color_type=0 (grayscale),
	// compression=0, filter=0, interlace=0
	ihdrData := []byte{
		0, 0, 0, 1, // width
		0, 0, 0, 1, // height
		8, 0, 0, 0, 0, // bit depth, color type, compression, filter, interlace
	}
	buf = appendChunk(buf, "IHDR", ihdrData)
	// IDAT: zlib-compressed single row (filter byte 0x00 + pixel 0x00)
	// Minimal valid zlib stream for [0x00, 0x00]:
	// 0x78 0x01 = zlib header (deflate, level 1)
	// 0x01 0x02 0x00 0xFD 0xFF = stored block, len=2
	// 0x00 0x00 = data
	// Adler32 of [0x00, 0x00] = 0x00010001
	idatData := []byte{
		0x78, 0x01, 0x01, 0x02, 0x00, 0xFD, 0xFF,
		0x00, 0x00,
		0x00, 0x01, 0x00, 0x01,
	}
	buf = appendChunk(buf, "IDAT", idatData)
	// IEND
	buf = appendChunk(buf, "IEND", nil)
	return buf
}

func appendChunk(buf []byte, typ string, data []byte) []byte {
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(data)))
	buf = append(buf, length...)
	buf = append(buf, typ...)
	buf = append(buf, data...)
	crc := crc32.NewIEEE()
	crc.Write([]byte(typ))
	crc.Write(data)
	crcBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBytes, crc.Sum32())
	buf = append(buf, crcBytes...)
	return buf
}

func TestPngHasSignature(t *testing.T) {
	if !pngHasSignature(minimalPNG()) {
		t.Error("expected true for valid PNG")
	}
	if pngHasSignature([]byte("not a png")) {
		t.Error("expected false for non-PNG data")
	}
	if pngHasSignature(nil) {
		t.Error("expected false for nil")
	}
	if pngHasSignature([]byte{0x89, 0x50}) {
		t.Error("expected false for truncated signature")
	}
}

func TestPngSetGetRoundTrip(t *testing.T) {
	png := minimalPNG()
	modified, err := pngSetText(png, "banana", `{"model":"flash"}`)
	if err != nil {
		t.Fatalf("pngSetText: %v", err)
	}

	// Modified should still be valid PNG
	if !pngHasSignature(modified) {
		t.Fatal("modified data lost PNG signature")
	}

	// Should be longer than original
	if len(modified) <= len(png) {
		t.Fatalf("modified (%d bytes) should be longer than original (%d bytes)", len(modified), len(png))
	}

	// Read it back
	val, err := pngGetText(modified, "banana")
	if err != nil {
		t.Fatalf("pngGetText: %v", err)
	}
	if val != `{"model":"flash"}` {
		t.Errorf("got %q, want %q", val, `{"model":"flash"}`)
	}
}

func TestPngSetTextRejectsNonPNG(t *testing.T) {
	_, err := pngSetText([]byte("not a png"), "key", "val")
	if err == nil {
		t.Fatal("expected error for non-PNG data")
	}
}

func TestPngGetTextRejectsNonPNG(t *testing.T) {
	_, err := pngGetText([]byte("not a png"), "key")
	if err == nil {
		t.Fatal("expected error for non-PNG data")
	}
}

func TestPngGetTextMissingKey(t *testing.T) {
	png := minimalPNG()
	modified, err := pngSetText(png, "other", "value")
	if err != nil {
		t.Fatalf("pngSetText: %v", err)
	}

	_, err = pngGetText(modified, "banana")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error %q should mention 'not found'", err)
	}
}

func TestPngMultipleTextChunks(t *testing.T) {
	png := minimalPNG()
	// Insert two different text chunks
	step1, err := pngSetText(png, "first", "one")
	if err != nil {
		t.Fatalf("first pngSetText: %v", err)
	}
	step2, err := pngSetText(step1, "second", "two")
	if err != nil {
		t.Fatalf("second pngSetText: %v", err)
	}

	val1, err := pngGetText(step2, "first")
	if err != nil {
		t.Fatalf("pngGetText first: %v", err)
	}
	if val1 != "one" {
		t.Errorf("first = %q, want %q", val1, "one")
	}

	val2, err := pngGetText(step2, "second")
	if err != nil {
		t.Fatalf("pngGetText second: %v", err)
	}
	if val2 != "two" {
		t.Errorf("second = %q, want %q", val2, "two")
	}
}
