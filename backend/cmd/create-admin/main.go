package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
	"github.com/alexandre/senshi-training-planner/backend/internal/config"
	"github.com/alexandre/senshi-training-planner/backend/internal/database"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	nameFlag := flag.String("name", "", "admin name")
	emailFlag := flag.String("email", "", "admin email")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)
	name, err := valueOrPrompt(reader, *nameFlag, "Name: ")
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}

	emailInput, err := valueOrPrompt(reader, *emailFlag, "Email: ")
	if err != nil {
		return err
	}
	email, err := auth.NormalizeEmail(emailInput)
	if err != nil {
		return err
	}

	password, err := promptPassword("Password: ")
	if err != nil {
		return err
	}
	confirmation, err := promptPassword("Confirm password: ")
	if err != nil {
		return err
	}
	if password != confirmation {
		return errors.New("password confirmation does not match")
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	userID, err := auth.NewUUID()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL, string(cfg.AppEnv))
	if err != nil {
		return err
	}
	defer pool.Close()

	store := auth.NewPostgresStore(pool)
	adminExists, err := store.AdminExists(ctx)
	if err != nil {
		return errors.New("could not check existing admins")
	}
	if adminExists {
		return errors.New("an admin account already exists")
	}

	emailExists, err := store.EmailExists(ctx, email)
	if err != nil {
		return errors.New("could not check existing email")
	}
	if emailExists {
		return errors.New("an account with this email already exists")
	}

	err = store.CreateAdmin(ctx, auth.User{
		ID:           userID,
		Name:         strings.TrimSpace(name),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         auth.RoleAdmin,
		Active:       true,
	})
	if err != nil {
		return errors.New("could not create admin")
	}

	fmt.Println("admin account created")
	return nil
}

func valueOrPrompt(reader *bufio.Reader, value string, prompt string) (string, error) {
	if value != "" {
		return value, nil
	}

	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(input), nil
}

func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}

	return string(password), nil
}
