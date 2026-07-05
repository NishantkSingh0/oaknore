package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/oaknore/pms3/internal/config"
	"github.com/oaknore/pms3/internal/database"
	"github.com/oaknore/pms3/internal/handlers"
	appmw "github.com/oaknore/pms3/internal/middleware"
	"github.com/oaknore/pms3/internal/models"
	"github.com/oaknore/pms3/internal/services"
	appws "github.com/oaknore/pms3/internal/websocket"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// ── Database ─────────────────────────────────────────
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	if err = database.RunMigrations(cfg.Database); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// ── AWS S3 ───────────────────────────────────────────
	s3Svc, err := services.NewS3Service(cfg.AWS)
	if err != nil {
		log.Fatalf("s3: %v", err)
	}

	// ── WebSocket Hub ─────────────────────────────────────
	hub := appws.NewHub(cfg.WS.PingInterval, cfg.WS.PongWait, cfg.WS.WriteWait)
	go hub.Run()

	// ── Handlers ─────────────────────────────────────────
	authH    := handlers.NewAuthHandler(db, cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)
	orgH     := handlers.NewOrgHandler(db)
	deptH    := handlers.NewDeptHandler(db)
	empH     := handlers.NewEmployeeHandler(db)
	projH    := handlers.NewProjectHandler(db, hub)
	fileH    := handlers.NewFileHandler(db, s3Svc)
	routingH := handlers.NewRoutingHandler(db, hub)
	taskH    := handlers.NewTaskHandler(db, hub)
	subH     := handlers.NewSubtaskHandler(db, hub)
	issueH   := handlers.NewIssueHandler(db, hub)
	reworkH  := handlers.NewReworkHandler(db, hub)
	matH     := handlers.NewMaterialHandler(db, hub)
	queryH   := handlers.NewQueryHandler(db, hub)
	reportH  := handlers.NewReportHandler(db, hub)
	notifH   := handlers.NewNotificationHandler(db)
	auditH   := handlers.NewAuditHandler(db)

	// ── Router ────────────────────────────────────────────
	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(appmw.Logger)
	r.Use(appmw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// ── WebSocket endpoint (authenticated) ───────────────
	r.Group(func(r chi.Router) {
		r.Use(appmw.Authenticate(cfg.JWT.Secret))
		r.Get("/ws", hub.ServeWS(cfg.JWT.Secret))
	})

	// ── API v1 ────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {

		// Public auth routes
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/refresh", authH.Refresh)
		r.Post("/auth/logout", authH.Logout)

		// All routes below require a valid JWT
		r.Group(func(r chi.Router) {
			r.Use(appmw.Authenticate(cfg.JWT.Secret))

			// Auth / self
			r.Get("/auth/me", authH.Me)
			r.Post("/auth/change-password", authH.ChangePassword)

			// ── Organization (Admin only) ─────────────
			r.Route("/org", func(r chi.Router) {
				r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin))
				r.Get("/", orgH.Get)
				r.Patch("/", orgH.Update)
			})

			// ── Departments ───────────────────────────
			r.Route("/departments", func(r chi.Router) {
				r.Get("/", deptH.List) // all authenticated users can list
				r.Group(func(r chi.Router) {
					r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin))
					r.Post("/", deptH.Create)
				})
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", deptH.GetOne)
					r.Group(func(r chi.Router) {
						r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin))
						r.Patch("/", deptH.Update)
					})
				})
			})

			// ── Employees ─────────────────────────────
			r.Route("/employees", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin))
					r.Get("/", empH.List)
					r.Post("/", empH.Create)
				})
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", empH.GetOne)
					r.Group(func(r chi.Router) {
						r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin))
						r.Patch("/", empH.Update)
						r.Post("/reset-password", empH.ResetPassword)
						r.Patch("/transfer", empH.Transfer)
					})
				})
			})

			// ── Projects ──────────────────────────────
			r.Route("/projects", func(r chi.Router) {
				r.Get("/", projH.List)
				r.Group(func(r chi.Router) {
					r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin))
					r.Post("/", projH.Create)
				})
				r.Route("/{projectID}", func(r chi.Router) {
					r.Get("/", projH.GetOne)
					r.Group(func(r chi.Router) {
						r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin))
						r.Patch("/", projH.Update)
						r.Patch("/status", projH.UpdateStatus)
					})
					r.Get("/revisions", projH.ListRevisions)
					r.Get("/timeline", projH.Timeline)

					// Routings (Layer 2 creates)
					r.Route("/routings", func(r chi.Router) {
						r.Get("/", routingH.List)
						r.Group(func(r chi.Router) {
							r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin, models.RoleLayer2))
							r.Post("/", routingH.Create)
						})
						r.Get("/{routingID}", routingH.GetOne)
					})

					// Tasks
					r.Get("/tasks", taskH.List)

					// Issues
					r.Route("/issues", func(r chi.Router) {
						r.Get("/", issueH.List)
						r.Post("/", issueH.Create)
					})

					// Reworks
					r.Route("/reworks", func(r chi.Router) {
						r.Get("/", reworkH.List)
						r.Post("/", reworkH.Create)
					})

					// Materials
					r.Route("/materials", func(r chi.Router) {
						r.Get("/", matH.List)
						r.Post("/", matH.Create)
					})

					// Queries
					r.Get("/queries", queryH.List)
					r.Post("/queries", queryH.Create)

					// Daily Reports
					r.Get("/reports", reportH.List)
					r.Post("/reports", reportH.Create)
				})
			})

			// ── Tasks (standalone) ─────────────────────
			r.Route("/tasks/{id}", func(r chi.Router) {
				r.Get("/", taskH.GetOne)
				r.Patch("/", taskH.Update)
				r.Patch("/status", taskH.UpdateStatus)
				r.Route("/subtasks", func(r chi.Router) {
					r.Post("/", subH.Create)
				})
			})

			// ── Subtasks (standalone) ─────────────────
			r.Route("/subtasks/{id}", func(r chi.Router) {
				r.Patch("/", subH.Update)
				r.Patch("/complete", subH.Complete)
				r.Delete("/", subH.Delete)
			})

			// ── Issues (standalone) ───────────────────
			r.Route("/issues/{id}", func(r chi.Router) {
				r.Get("/", issueH.GetOne)
				r.Group(func(r chi.Router) {
					r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin, models.RoleLayer2))
					r.Patch("/review", issueH.Review)
				})
				r.Group(func(r chi.Router) {
					r.Use(appmw.RequireRoles(models.RoleLayer3))
					r.Patch("/resolve", issueH.Resolve)
				})
			})

			// ── Reworks (standalone) ──────────────────
			r.Route("/reworks/{id}", func(r chi.Router) {
				r.Get("/", reworkH.GetOne)
				r.Group(func(r chi.Router) {
					r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin, models.RoleLayer2))
					r.Patch("/review", reworkH.Review)
				})
			})

			// ── Materials (standalone) ────────────────
			r.Route("/materials/{id}", func(r chi.Router) {
				r.Get("/", matH.GetOne)
				r.Group(func(r chi.Router) {
					r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin, models.RoleLayer2))
					r.Patch("/review", matH.Review)
					r.Patch("/fulfill", matH.Fulfill)
				})
			})

			// ── Queries (standalone) ──────────────────
			r.Route("/queries/{id}", func(r chi.Router) {
				r.Get("/", queryH.GetOne)
				r.Post("/messages", queryH.PostMessage)
				r.Patch("/resolve", queryH.Resolve)
			})

			// ── Reports (standalone) ──────────────────
			r.Get("/reports/{id}", reportH.GetOne)

			// ── Files ─────────────────────────────────
			r.Post("/files/upload", fileH.Upload)
			r.Get("/files", fileH.ListByOwner)
			r.Get("/files/{id}/presign", fileH.Presign)
			r.Delete("/files/{id}", fileH.Delete)

			// ── Notifications ─────────────────────────
			r.Get("/notifications", notifH.List)
			r.Get("/notifications/unread-count", notifH.UnreadCount)
			r.Patch("/notifications/read-all", notifH.MarkAllRead)
			r.Patch("/notifications/{id}/read", notifH.MarkRead)

			// ── Audit (Admin only) ────────────────────
			r.Group(func(r chi.Router) {
				r.Use(appmw.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin))
				r.Get("/audit", auditH.List)
			})
		})
	})

	addr := ":" + cfg.App.Port
	log.Printf("PMS3 API listening on %s (env=%s)", addr, cfg.App.Env)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
