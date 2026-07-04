package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/oaknore/pms3/internal/config"
)

type SuperAdmin struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
	Phone     string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := sqlx.Open("postgres", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// Create organization
	orgID := uuid.New()
	orgSlug := "oak-nore"
	orgName := "Oak & ORE"

	var existingOrg string
	err = db.Get(&existingOrg, "SELECT id FROM organizations WHERE slug=$1", orgSlug)
	if err == nil {
		fmt.Printf("Organization '%s' already exists (ID: %s)\n", orgSlug, existingOrg)
		orgID, _ = uuid.Parse(existingOrg)
	} else {
		_, err = db.Exec(`
			INSERT INTO organizations (id, name, slug)
			VALUES ($1, $2, $3)
		`, orgID, orgName, orgSlug)
		if err != nil {
			log.Fatalf("create organization: %v", err)
		}
		fmt.Printf("Created organization: %s (ID: %s)\n", orgName, orgID)
	}

	// SuperAdmins to seed
	superAdmins := []SuperAdmin{
		{
			FirstName: "Nishant",
			LastName:  "Singh",
			Email:     "n@oaknore.in",
			Password:  "O$1234567890",
			Phone:     "9917760469",
		},
		// {
		// 	FirstName: "Admin",
		// 	LastName:  "User",
		// 	Email:     "admin@oaknore.com",
		// 	Password:  "admin123",
		// 	Phone:     "9876543211",
		// },
	}

	fmt.Println("\nSeeding SuperAdmins...")
	for _, sa := range superAdmins {
		// Check if user already exists
		var existingID string
		err = db.Get(&existingID, "SELECT id FROM users WHERE email=$1", sa.Email)
		if err == nil {
			fmt.Printf("User '%s' already exists (ID: %s) - skipping\n", sa.Email, existingID)
			continue
		}

		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(sa.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("password hashing failed for %s: %v", sa.Email, err)
		}

		userID := uuid.New()
		_, err = db.Exec(`
			INSERT INTO users (id, org_id, first_name, last_name, email, phone, password_hash, role)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'SUPER_ADMIN')
		`, userID, orgID, sa.FirstName, sa.LastName, sa.Email, sa.Phone, string(hash))
		if err != nil {
			log.Fatalf("create user %s: %v", sa.Email, err)
		}

		fmt.Printf("Created SuperAdmin: %s %s (%s)\n", sa.FirstName, sa.LastName, sa.Email)
	}

	fmt.Println("\n✅ Seed completed successfully!")
}
