package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// CategoryItem represents a category configuration item
type CategoryItem struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	English      string `json:"english"`
}

// ColorItem represents a color configuration item
type ColorItem struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	English      string `json:"english"`
}

// AccountItem represents an account configuration item
type AccountItem struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AccountNumber string `json:"accountNumber"`
	BankName      string `json:"bankName"`
	AccountType   string `json:"accountType"`
	IsActive      bool   `json:"isActive"`
}

// ConfigData holds all configuration data
type ConfigData struct {
	Categories []CategoryItem `json:"categories"`
	Colors     []ColorItem    `json:"colors"`
	Accounts   []AccountItem  `json:"accounts"`
}

// ConfigLoader handles loading configuration from JSON files
type ConfigLoader struct {
	config ConfigData
}

// NewConfigLoader creates a new config loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

// LoadConfig loads configuration from JSON files
func (cl *ConfigLoader) LoadConfig() error {
	// Get the directory where the executable is located
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)

	// Try to find config files in the same directory as the executable
	configDir := filepath.Join(execDir, "config")

	// If config directory doesn't exist, try relative to current working directory
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		configDir = "config"
	}

	// Load categories
	if err := cl.loadCategories(filepath.Join(configDir, "categories.json")); err != nil {
		return fmt.Errorf("failed to load categories: %v", err)
	}

	// Load colors
	if err := cl.loadColors(filepath.Join(configDir, "colors.json")); err != nil {
		return fmt.Errorf("failed to load colors: %v", err)
	}

	// Load accounts
	if err := cl.loadAccounts(filepath.Join(configDir, "accounts.json")); err != nil {
		return fmt.Errorf("failed to load accounts: %v", err)
	}

	return nil
}

// loadCategories loads categories from JSON file
func (cl *ConfigLoader) loadCategories(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var categories struct {
		Categories []CategoryItem `json:"categories"`
	}

	if err := json.Unmarshal(data, &categories); err != nil {
		return err
	}

	cl.config.Categories = categories.Categories
	return nil
}

// loadColors loads colors from JSON file
func (cl *ConfigLoader) loadColors(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var colors struct {
		Colors []ColorItem `json:"colors"`
	}

	if err := json.Unmarshal(data, &colors); err != nil {
		return err
	}

	cl.config.Colors = colors.Colors
	return nil
}

// loadAccounts loads accounts from JSON file
func (cl *ConfigLoader) loadAccounts(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var accounts []AccountItem
	if err := json.Unmarshal(data, &accounts); err != nil {
		return err
	}

	cl.config.Accounts = accounts
	return nil
}

// GetCategories returns all categories
func (cl *ConfigLoader) GetCategories() []CategoryItem {
	return cl.config.Categories
}

// GetColors returns all colors
func (cl *ConfigLoader) GetColors() []ColorItem {
	return cl.config.Colors
}

// GetAccounts returns all accounts
func (cl *ConfigLoader) GetAccounts() []AccountItem {
	return cl.config.Accounts
}

// GetActiveAccounts returns only active accounts
func (cl *ConfigLoader) GetActiveAccounts() []AccountItem {
	var activeAccounts []AccountItem
	for _, account := range cl.config.Accounts {
		if account.IsActive {
			activeAccounts = append(activeAccounts, account)
		}
	}
	return activeAccounts
}

// GetCategoryAbbreviation returns abbreviation for a category name
func (cl *ConfigLoader) GetCategoryAbbreviation(categoryName string) string {
	categoryLower := strings.ToLower(categoryName)

	for _, category := range cl.config.Categories {
		if strings.ToLower(category.Name) == categoryLower ||
			strings.ToLower(category.English) == categoryLower {
			return category.Abbreviation
		}
	}

	// If not found, create abbreviation from first 2-3 characters
	words := strings.Fields(categoryName)
	if len(words) == 1 {
		if len(categoryName) >= 3 {
			return strings.ToUpper(categoryName[:3])
		}
		return strings.ToUpper(categoryName)
	} else {
		abbrev := ""
		for _, word := range words {
			if len(word) > 0 {
				abbrev += strings.ToUpper(string(word[0]))
			}
		}
		if len(abbrev) > 3 {
			abbrev = abbrev[:3]
		}
		return abbrev
	}
}

// GetColorAbbreviation returns abbreviation for a color name
func (cl *ConfigLoader) GetColorAbbreviation(colorName string) string {
	colorLower := strings.ToLower(colorName)

	for _, color := range cl.config.Colors {
		if strings.ToLower(color.Name) == colorLower ||
			strings.ToLower(color.English) == colorLower {
			return color.Abbreviation
		}
	}

	// If not found, create abbreviation from first 2 characters
	if len(colorName) >= 2 {
		return strings.ToUpper(colorName[:2])
	}
	return strings.ToUpper(colorName)
}
