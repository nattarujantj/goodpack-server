package utils

import (
	"regexp"
	"strings"
)

// NormalizeSize normalizes size strings to standard format
// Examples:
//   - "50 ml." → "50 mL"
//   - "50ml." → "50 mL"
//   - "50ML" → "50 mL"
//   - "50ML." → "50 mL"
//   - "1l" → "1 L"
//   - "1L." → "1 L"
//   - "500cc" → "500 cc"
func NormalizeSize(size string) string {
	if size == "" {
		return size
	}

	// Trim spaces
	result := strings.TrimSpace(size)

	// Remove trailing dots
	result = strings.TrimSuffix(result, ".")

	// Pattern: number followed by optional space and unit
	// Captures: (number)(optional space)(unit)
	re := regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(ml|l|cc|g|kg|oz|mm|cm|m)\.?$`)

	matches := re.FindStringSubmatch(result)
	if len(matches) >= 3 {
		number := matches[1]
		unit := strings.ToLower(matches[2])

		// Normalize unit
		normalizedUnit := normalizeUnit(unit)

		return number + " " + normalizedUnit
	}

	// If no match, just clean up the string
	result = strings.TrimSuffix(result, ".")
	return result
}

// normalizeUnit converts unit to standard format
func normalizeUnit(unit string) string {
	unit = strings.ToLower(unit)

	switch unit {
	case "ml":
		return "mL" // มิลลิลิตร - ใช้ L ตัวใหญ่
	case "l":
		return "L" // ลิตร - ใช้ L ตัวใหญ่
	case "cc":
		return "cc" // ซีซี
	case "g":
		return "g" // กรัม
	case "kg":
		return "kg" // กิโลกรัม
	case "oz":
		return "oz" // ออนซ์
	case "mm":
		return "mm" // มิลลิเมตร
	case "cm":
		return "cm" // เซนติเมตร
	case "m":
		return "m" // เมตร
	default:
		return unit
	}
}

