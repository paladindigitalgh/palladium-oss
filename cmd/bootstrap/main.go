// Command bootstrap creates the first administrator account for a new
// Palladium installation. It is a separate binary from the API server for
// the same reason cmd/migrate is: creating the initial account is an
// explicit, one-time, by-hand installation step, not something the server
// should ever do implicitly on boot.
//
// This tool is for initial installation only. It refuses to run if any
// user account already exists (see internal/auth/bootstrap, which
// contains that check and the actual account-creation logic — this file
// is only the interactive terminal wrapper around it) and it does not
// implement general user management: no listing, editing, or deleting
// users, no roles, no permissions. Those are separate, later concerns.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/paladindigitalgh/palladium-oss/internal/auth/bootstrap"
	authpostgres "github.com/paladindigitalgh/palladium-oss/internal/auth/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/config"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Connect before prompting: if the database is unreachable, fail
	// immediately rather than asking someone to type a password first and
	// only then discovering the tool can't do anything with it.
	pool, err := database.Connect(ctx, database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	email, err := promptEmail()
	if err != nil {
		return err
	}

	password, err := promptPassword()
	if err != nil {
		return err
	}

	// clock.New() and id.New() match how every other repository in this
	// codebase is constructed (see e.g. cmd/server/main.go): they are
	// stateless, so there is no reason for this one-off tool to do
	// anything differently. id.New() is "the existing ID generator" this
	// milestone asks for — it is used inside UserRepository.Create itself
	// (see internal/auth/postgres/user.go), not called separately here.
	users := authpostgres.NewUserRepository(pool, clock.New(), id.New())
	admin := bootstrap.NewAdministrator(users)

	user, err := admin.Create(ctx, email, password)
	if err != nil {
		return err
	}

	fmt.Printf("administrator account created: %s\n", user.Email)
	return nil
}

func promptEmail() (string, error) {
	fmt.Print("Email: ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read email: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// promptPassword reads a password from the terminal with echo disabled,
// via golang.org/x/term — the standard, Go-team-maintained package for
// exactly this (the same family as golang.org/x/crypto, already used by
// internal/auth/password.go). Hand-rolling raw terminal mode via direct
// termios syscalls would be needless, error-prone platform-specific code
// for a solved problem.
func promptPassword() (string, error) {
	fmt.Print("Password: ")

	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // term.ReadPassword does not echo the Enter keystroke
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(password), nil
}
