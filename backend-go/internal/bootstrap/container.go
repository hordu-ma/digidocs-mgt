package bootstrap

import (
	"database/sql"
	"log"
	"path/filepath"
	"runtime"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/db"
	"digidocs-mgt/backend-go/internal/queue"
	memqueue "digidocs-mgt/backend-go/internal/queue/memory"
	noopqueue "digidocs-mgt/backend-go/internal/queue/noop"
	pgqueue "digidocs-mgt/backend-go/internal/queue/postgres"
	"digidocs-mgt/backend-go/internal/repository/memory"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/storage"
	memstorage "digidocs-mgt/backend-go/internal/storage/memory"
	synostorage "digidocs-mgt/backend-go/internal/storage/synology"
)

type Container struct {
	DB                    *sql.DB
	QueueConsumer         queue.Consumer
	QueryService          service.QueryService
	AssistantService      service.AssistantService
	DocumentService       service.DocumentService
	AuditQueryService     service.AuditQueryService
	DashboardQueryService service.DashboardQueryService
	VersionService        service.VersionService
	FlowService           service.FlowService
	HandoverService       service.HandoverService
	AuthService           service.AuthService
	TokenService          service.TokenService
	AuditService          service.AuditService
}

func BuildContainer(cfg config.Config) (Container, error) {
	publisher := memqueue.NewPublisher()
	storageProvider := buildStorageProvider(cfg)
	tokenService := service.NewTokenService(cfg.JWTSecret)
	auditService := service.NewAuditService()

	switch cfg.DataBackend {
	case "postgres":
		postgresDB, err := db.OpenPostgres(cfg.DatabaseURL)
		if err != nil {
			return Container{}, err
		}

		migrationsDir := findMigrationsDir()
		if migrationsDir != "" {
			if err := db.RunMigrations(postgresDB, migrationsDir); err != nil {
				log.Printf("[bootstrap] migration warning: %v", err)
			}
		}

		actionRepo := pgrepo.NewActionRepository(postgresDB)
		authService := service.NewAuthService(pgrepo.NewUserAuthRepository(postgresDB), tokenService)
		versionRepo := pgrepo.NewVersionRepository(postgresDB)
		docRepo := pgrepo.NewDocumentRepository(postgresDB)
		versionWorkflow := pgrepo.NewVersionWorkflow(postgresDB)

		return Container{
			DB:            postgresDB,
			QueueConsumer: pgqueue.NewConsumer(postgresDB),
			QueryService: service.NewQueryService(
				pgrepo.NewTeamSpaceRepository(postgresDB),
				pgrepo.NewProjectRepository(postgresDB),
			),
			AssistantService:      service.NewAssistantService(noopqueue.NewPublisher(), pgrepo.NewAssistantRepository(postgresDB)),
			DocumentService:       service.NewDocumentService(docRepo, docRepo, storageProvider, versionWorkflow),
			AuditQueryService:     service.NewAuditQueryService(pgrepo.NewAuditRepository(postgresDB)),
			DashboardQueryService: service.NewDashboardQueryService(pgrepo.NewDashboardRepository(postgresDB)),
			VersionService:        service.NewVersionService(storageProvider, versionWorkflow, versionRepo),
			FlowService:           service.NewFlowService(pgrepo.NewFlowRepository(postgresDB), actionRepo),
			HandoverService:       service.NewHandoverService(pgrepo.NewHandoverRepository(postgresDB), actionRepo),
			AuthService:           authService,
			TokenService:          tokenService,
			AuditService:          auditService,
		}, nil
	default:
		actionRepo := memory.NewActionRepository()
		authService := service.NewAuthService(memory.NewUserAuthRepository(), tokenService)
		docRepo := memory.NewDocumentRepository()
		return Container{
			QueueConsumer: publisher,
			QueryService: service.NewQueryService(
				memory.NewTeamSpaceRepository(),
				memory.NewProjectRepository(),
			),
			AssistantService:      service.NewAssistantService(publisher, memory.NewAssistantRepository()),
			DocumentService:       service.NewDocumentService(docRepo, docRepo, storageProvider, memory.NewVersionWorkflow()),
			AuditQueryService:     service.NewAuditQueryService(memory.NewAuditRepository()),
			DashboardQueryService: service.NewDashboardQueryService(memory.NewDashboardRepository()),
			VersionService:        service.NewVersionService(storageProvider, memory.NewVersionWorkflow(), memory.NewVersionRepository()),
			FlowService:           service.NewFlowService(memory.NewFlowRepository(), actionRepo),
			HandoverService:       service.NewHandoverService(memory.NewHandoverRepository(), actionRepo),
			AuthService:           authService,
			TokenService:          tokenService,
			AuditService:          auditService,
		}, nil
	}
}

// findMigrationsDir locates the migrations/ directory relative to the project root.
func findMigrationsDir() string {
	// Try relative to working directory first.
	candidates := []string{"migrations", "backend-go/migrations"}

	// Also try relative to this source file (for tests).
	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations"))
	}

	for _, dir := range candidates {
		if info, err := filepath.Glob(filepath.Join(dir, "*.sql")); err == nil && len(info) > 0 {
			abs, _ := filepath.Abs(dir)
			return abs
		}
	}

	return ""
}

// buildStorageProvider returns the storage.Provider based on configuration.
func buildStorageProvider(cfg config.Config) storage.Provider {
	switch cfg.StorageBackend {
	case "synology":
		if cfg.SynologyHost == "" {
			log.Fatal("FATAL: SYNOLOGY_HOST is required when STORAGE_BACKEND=synology")
		}
		log.Printf("[bootstrap] using Synology storage: %s:%d share=%s",
			cfg.SynologyHost, cfg.SynologyPort, cfg.SynologySharePath)
		return synostorage.NewProvider(synostorage.Config{
			Host:      cfg.SynologyHost,
			Port:      cfg.SynologyPort,
			HTTPS:     cfg.SynologyHTTPS,
			Account:   cfg.SynologyAccount,
			Password:  cfg.SynologyPassword,
			SharePath: cfg.SynologySharePath,
		})
	default:
		log.Println("[bootstrap] using memory storage")
		return memstorage.NewProvider()
	}
}
