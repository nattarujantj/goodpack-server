package utils

import (
	"time"
)

// Thailand timezone (GMT+7)
var ThailandLocation *time.Location

func init() {
	var err error
	ThailandLocation, err = time.LoadLocation("Asia/Bangkok")
	if err != nil {
		// Fallback to fixed timezone if Asia/Bangkok is not available
		ThailandLocation = time.FixedZone("GMT+7", 7*60*60)
	}
}

// NowInThailand returns current time in Thailand timezone
func NowInThailand() time.Time {
	return time.Now().In(ThailandLocation)
}

// ToThailand converts a time to Thailand timezone
func ToThailand(t time.Time) time.Time {
	return t.In(ThailandLocation)
}

// ParseInThailand parses a date string and returns time in Thailand timezone
func ParseInThailand(layout, value string) (time.Time, error) {
	t, err := time.ParseInLocation(layout, value, ThailandLocation)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

