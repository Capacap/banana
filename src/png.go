package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"

	_ "image/jpeg"
)

var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

func pngHasSignature(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	for i := 0; i < 8; i++ {
		if data[i] != pngSignature[i] {
			return false
		}
	}
	return true
}

// pngSetText inserts a tEXt chunk with the given key and value after the IHDR chunk.
// The input must be a valid PNG (starts with the 8-byte signature followed by IHDR).
func pngSetText(data []byte, key, value string) ([]byte, error) {
	if !pngHasSignature(data) {
		return nil, errors.New("not a PNG file")
	}

	// IHDR starts at offset 8. Read its data length to find where it ends.
	if len(data) < 8+8 { // signature + length + type minimum
		return nil, errors.New("PNG too short")
	}
	ihdrLen := binary.BigEndian.Uint32(data[8:12])
	// End of IHDR chunk: signature(8) + length(4) + type(4) + data(ihdrLen) + crc(4)
	insertAt := 8 + 4 + 4 + int(ihdrLen) + 4
	if insertAt > len(data) {
		return nil, errors.New("PNG IHDR extends beyond data")
	}

	// Build tEXt chunk payload: key + null separator + value
	payload := make([]byte, len(key)+1+len(value))
	copy(payload, key)
	payload[len(key)] = 0x00
	copy(payload[len(key)+1:], value)

	// Build the full chunk: length(4) + "tEXt"(4) + payload + CRC(4)
	chunkType := []byte("tEXt")
	chunk := make([]byte, 4+4+len(payload)+4)
	binary.BigEndian.PutUint32(chunk[0:4], uint32(len(payload)))
	copy(chunk[4:8], chunkType)
	copy(chunk[8:], payload)
	crc := crc32.NewIEEE()
	crc.Write(chunkType)
	crc.Write(payload)
	binary.BigEndian.PutUint32(chunk[8+len(payload):], crc.Sum32())

	// Splice: data[:insertAt] + chunk + data[insertAt:]
	result := make([]byte, len(data)+len(chunk))
	copy(result, data[:insertAt])
	copy(result[insertAt:], chunk)
	copy(result[insertAt+len(chunk):], data[insertAt:])

	return result, nil
}

// ensurePNG returns the data unchanged if it is already PNG. Otherwise it decodes
// the image (JPEG, etc.) and re-encodes it as PNG.
func ensurePNG(data []byte) ([]byte, error) {
	if pngHasSignature(data) {
		return data, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image data: %v", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode as PNG: %v", err)
	}
	return buf.Bytes(), nil
}

// pngGetText scans a PNG for a tEXt chunk matching the given key and returns its value.
func pngGetText(data []byte, key string) (string, error) {
	if !pngHasSignature(data) {
		return "", errors.New("not a PNG file")
	}

	offset := 8 // skip signature
	for offset+8 <= len(data) {
		chunkLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		chunkType := string(data[offset+4 : offset+8])

		chunkEnd := offset + 8 + chunkLen + 4 // length + type + data + CRC
		if chunkEnd > len(data) {
			break
		}

		if chunkType == "tEXt" {
			payload := data[offset+8 : offset+8+chunkLen]
			// Find null separator between key and value
			for i := 0; i < len(payload); i++ {
				if payload[i] == 0x00 {
					if string(payload[:i]) == key {
						return string(payload[i+1:]), nil
					}
					break
				}
			}
		}

		offset = chunkEnd
	}

	return "", errors.New("tEXt chunk not found for key: " + key)
}
