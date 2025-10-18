package services

import (
	"encoding/json"
	"os"
	"path/filepath"

	"goodpack-server/models"
)

type BankAccountService struct{}

func NewBankAccountService() *BankAccountService {
	return &BankAccountService{}
}

// LoadBankAccountFromConfig loads bank account from config file
func (s *BankAccountService) LoadBankAccountFromConfig(accountID string) (*models.BankAccount, error) {
	// Get the directory of the current file
	dir := filepath.Dir(".")
	configPath := filepath.Join(dir, "config", "accounts.json")

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var accounts []models.BankAccount
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, err
	}

	// Find the account by ID
	for _, account := range accounts {
		if account.ID == accountID {
			return &account, nil
		}
	}

	return nil, nil // Account not found
}

// LoadAllBankAccounts loads all bank accounts from config file
func (s *BankAccountService) LoadAllBankAccounts() ([]models.BankAccount, error) {
	// Get the directory of the current file
	dir := filepath.Dir(".")
	configPath := filepath.Join(dir, "config", "accounts.json")

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var accounts []models.BankAccount
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}
