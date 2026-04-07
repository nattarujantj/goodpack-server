package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"goodpack-server/config"
	"goodpack-server/database"
	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/utils"
)

func main() {
	username := flag.String("username", "", "Username for the super admin")
	password := flag.String("password", "", "Password for the super admin")
	displayName := flag.String("display-name", "", "Display name for the super admin")
	flag.Parse()

	if *username == "" || *password == "" {
		log.Fatal("Usage: go run cmd/seed/main.go --username <username> --password <password> [--display-name <name>]")
	}

	if *displayName == "" {
		*displayName = *username
	}

	cfg := config.Load()

	mongoDB, err := database.NewMongoDB(cfg.MongoURI, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoDB.Close()

	userRepo := repository.NewUserRepository(mongoDB.GetCollection("users"))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	existing, _ := userRepo.FindByUsername(ctx, *username)
	if existing != nil {
		log.Fatalf("User '%s' already exists", *username)
	}

	now := utils.NowInThailand()
	user := &models.User{
		Username:    *username,
		DisplayName: *displayName,
		Role:        models.RoleSuperAdmin,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := user.SetPassword(*password); err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	if err := userRepo.Create(ctx, user); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("Super admin '%s' created successfully (ID: %s)\n", *username, user.ID.Hex())
}
