package utils

import (
	"bytes"
	"io"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// ConvertToUTF8 converts file bytes from various encodings (Windows-874, TIS-620, etc.) to UTF-8
func ConvertToUTF8(data []byte) ([]byte, error) {
	// Check if already UTF-8 with BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		// Remove BOM and return
		return data[3:], nil
	}

	// Try to detect if it's already valid UTF-8
	if IsUTF8(data) {
		return data, nil
	}

	// Try Windows-874 (Thai Windows encoding)
	decoder := charmap.Windows874.NewDecoder()
	utf8Bytes, err := io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
	if err == nil && IsUTF8(utf8Bytes) {
		return utf8Bytes, nil
	}

	// If all else fails, try to read as UTF-8 anyway (might have some corruption)
	return data, nil
}

// IsUTF8 checks if the byte slice is valid UTF-8
func IsUTF8(data []byte) bool {
	for i := 0; i < len(data); {
		if data[i] < 0x80 {
			i++
		} else if data[i] < 0xE0 {
			if i+1 >= len(data) || (data[i+1]&0xC0) != 0x80 {
				return false
			}
			i += 2
		} else if data[i] < 0xF0 {
			if i+2 >= len(data) || (data[i+1]&0xC0) != 0x80 || (data[i+2]&0xC0) != 0x80 {
				return false
			}
			i += 3
		} else {
			if i+3 >= len(data) || (data[i+1]&0xC0) != 0x80 || (data[i+2]&0xC0) != 0x80 || (data[i+3]&0xC0) != 0x80 {
				return false
			}
			i += 4
		}
	}
	return true
}


