package utils

import "math"

// RoundTo2 rounds a float64 to 2 decimal places.
func RoundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}
