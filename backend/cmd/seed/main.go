package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/oaknore/pms3/internal/config"
	"github.com/oaknore/pms3/internal/database"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	// ── Create org ────────────────────────────────────────
	orgID := uuid.New()
	_, err = db.Exec(
		`INSERT INTO organizations(id, name, slug)
		 VALUES($1, $2, $3)
		 ON CONFLICT (slug) DO UPDATE SET name=EXCLUDED.name
		 RETURNING id`,
		orgID, "OAKnORE", "oaknore",
	)
	if err != nil {
		// org likely already exists — fetch its id
		if e := db.Get(&orgID, `SELECT id FROM organizations WHERE slug='oaknore'`); e != nil {
			log.Fatalf("org: %v / %v", err, e)
		}
		log.Printf("org already exists: %s", orgID)
	} else {
		log.Printf("org created: %s", orgID)
	}

	// ── Create SUPER_ADMIN user ───────────────────────────
	hash, _ := bcrypt.GenerateFromPassword([]byte("Admin@123"), bcrypt.DefaultCost)
	adminID := uuid.New()
	_, err = db.Exec(
		`INSERT INTO users(id, org_id, first_name, last_name, email, password_hash, role)
		 VALUES($1, $2, 'Admin', 'User', 'admin@pms3.com', $3, 'SUPER_ADMIN')
		 ON CONFLICT (email) DO UPDATE SET password_hash=EXCLUDED.password_hash, org_id=EXCLUDED.org_id`,
		adminID, orgID, string(hash),
	)
	if err != nil {
		log.Fatalf("admin user: %v", err)
	}

	// ── Create sample Layer 2 department ─────────────────
	l2ID := uuid.New()
	_, _ = db.Exec(
		`INSERT INTO departments(id, org_id, name, layer)
		 VALUES($1, $2, 'Production Management', 'LAYER_2')
		 ON CONFLICT DO NOTHING`,
		l2ID, orgID,
	)

	// ── Create sample Layer 3 departments ────────────────
	for _, name := range []string{"Metal", "Carpentry", "Upholstery", "Finishing", "Quality"} {
		_, _ = db.Exec(
			`INSERT INTO departments(id, org_id, name, layer)
			 VALUES($1, $2, $3, 'LAYER_3')
			 ON CONFLICT DO NOTHING`,
			uuid.New(), orgID, name,
		)
	}

	fmt.Println("──────────────────────────────────────────")
	fmt.Println("Seed complete.")
	fmt.Printf("  Org ID   : %s\n", orgID)
	fmt.Println("  Login    : admin@pms3.com")
	fmt.Println("  Password : Admin@123")
	fmt.Println("──────────────────────────────────────────")
}
