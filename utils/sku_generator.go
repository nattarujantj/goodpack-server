package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"goodpack-server/config"
)

// SKUGenerator handles SKU ID generation
type SKUGenerator struct {
	configLoader *config.ConfigLoader
}

// NewSKUGenerator creates a new SKU generator
func NewSKUGenerator() *SKUGenerator {
	configLoader := config.NewConfigLoader()
	if err := configLoader.LoadConfig(); err != nil {
		// If config loading fails, continue with empty config
		fmt.Printf("Warning: Failed to load config: %v\n", err)
	}

	return &SKUGenerator{
		configLoader: configLoader,
	}
}

// GenerateSKUID generates a SKU ID based on category
// Format: XY-0000 or XYZ-0000 (depending on category abbreviation length)
func (sg *SKUGenerator) GenerateSKUID(category string, lastNumber int) string {
	// Get category abbreviation
	abbrev := sg.getCategoryAbbreviation(category)

	// Generate next number
	nextNumber := lastNumber + 1

	// Format with leading zeros
	format := fmt.Sprintf("%s-%%04d", abbrev)
	return fmt.Sprintf(format, nextNumber)
}

// GenerateProductCode generates a product code
// Format: XY-aaaa/AB (XY=category abbrev, aaaa=size, AB=color abbrev)
func (sg *SKUGenerator) GenerateProductCode(category, size, color string) string {
	categoryAbbrev := sg.getCategoryAbbreviation(category)
	colorAbbrev := sg.getColorAbbreviation(color)

	// Format size (remove spaces, convert to lowercase)
	formattedSize := strings.ReplaceAll(strings.ToLower(size), " ", "")

	return fmt.Sprintf("%s-%s/%s", categoryAbbrev, formattedSize, colorAbbrev)
}

// getCategoryAbbreviation returns abbreviation for category
func (sg *SKUGenerator) getCategoryAbbreviation(category string) string {
	return sg.configLoader.GetCategoryAbbreviation(category)
}

// getColorAbbreviation returns abbreviation for color
func (sg *SKUGenerator) getColorAbbreviation(color string) string {
	return sg.configLoader.GetColorAbbreviation(color)
}

// ParseSKUID extracts category and number from SKU ID
func (sg *SKUGenerator) ParseSKUID(skuID string) (category string, number int, err error) {
	// Regex pattern for SKU ID: XY-0000 or XYZ-0000
	pattern := `^([A-Z]{2,3})-(\d{4})$`
	regex := regexp.MustCompile(pattern)

	matches := regex.FindStringSubmatch(skuID)
	if len(matches) != 3 {
		return "", 0, fmt.Errorf("invalid SKU ID format: %s", skuID)
	}

	category = matches[1]
	number, err = strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid number in SKU ID: %s", skuID)
	}

	return category, number, nil
}

// GetNextSKUNumber gets the next number for a category
func (sg *SKUGenerator) GetNextSKUNumber(category string, existingSKUs []string) int {
	categoryAbbrev := sg.getCategoryAbbreviation(category)
	maxNumber := 0

	for _, sku := range existingSKUs {
		parsedCategory, number, err := sg.ParseSKUID(sku)
		if err != nil {
			continue
		}

		if parsedCategory == categoryAbbrev && number > maxNumber {
			maxNumber = number
		}
	}

	return maxNumber
}
