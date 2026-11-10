package bootstrap

import (
	"database/sql"
	"fmt"
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
	AdminService          service.AdminService
	AssistantService      service.AssistantService
	DocumentService       service.DocumentService
	AuditQueryService     service.AuditQueryService
	DashboardQueryService service.DashboardQueryService
	VersionService        service.VersionService
	FlowService           service.FlowService
	HandoverService       service.HandoverService
	DataAssetService      service.DataAssetService
	CodeRepositoryService service.CodeRepositoryService
	AuthService           service.AuthService
	TokenService          service.TokenService
	AuditService          service.AuditService
	MembershipService     service.MembershipService
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

		// Migrations are a hard startup dependency, not a best-effort step.
		//
		// This used to log a warning and carry on, which meant a failed or
		// skipped migration left the process serving traffic against a stale or
		// half-applied schema — surfacing later as scattered 500s or, worse, as
		// writes that silently land in the wrong shape. Nobody reads a warning
		// in the boot log. Refusing to start is the safer failure: the container
		// restart policy will retry, and a genuinely broken migration becomes
		// immediately visible instead of quietly corrupting data.
		migrationsDir := findMigrationsDir()
		if migrationsDir == "" {
			return Container{}, fmt.Errorf("migrations directory not found; refusing to start against an unmigrated database")
		}
		if err := db.RunMigrations(postgresDB, migrationsDir); err != nil {
			return Container{}, fmt.Errorf("run migrations: %w", err)
		}
		log.Printf("[bootstrap] migrations applied from %s", migrationsDir)

		actionRepo := pgrepo.NewActionRepository(postgresDB)
		authService := service.NewAuthService(pgrepo.NewUserAuthRepository(postgresDB), tokenService)
		versionRepo := pgrepo.NewVersionRepository(postgresDB)
		docRepo := pgrepo.NewDocumentRepository(postgresDB)
		versionWorkflow := pgrepo.NewVersionWorkflow(postgresDB)
		permissionService := service.NewPermissionService(pgrepo.NewPermissionRepository(postgresDB))

		dataAssetRepo := pgrepo.NewDataAssetRepository(postgresDB)
		codeRepo := pgrepo.NewCodeRepositoryRepository(postgresDB)
		auditService := service.NewAuditService(pgrepo.NewAuditRepository(postgresDB))
		return Container{
			DB:            postgresDB,
			QueueConsumer: pgqueue.NewConsumer(postgresDB),
			QueryService: service.NewQueryService(
				pgrepo.NewTeamSpaceRepository(postgresDB),
				pgrepo.NewUserQueryRepository(postgresDB),
				pgrepo.NewProjectRepository(postgresDB),
			),
			AdminService:          service.NewAdminService(pgrepo.NewAdminRepository(postgresDB)),
			AssistantService:      service.NewAssistantService(noopqueue.NewPublisher(), pgrepo.NewAssistantRepository(postgresDB), docRepo, versionRepo, permissionService),
			DocumentService:       service.NewDocumentService(docRepo, docRepo, storageProvider, versionWorkflow, permissionService),
			AuditQueryService:     service.NewAuditQueryService(pgrepo.NewAuditRepository(postgresDB)),
			DashboardQueryService: service.NewDashboardQueryService(pgrepo.NewDashboardRepository(postgresDB)),
			VersionService:        service.NewVersionService(storageProvider, versionWorkflow, versionRepo, permissionService),
			FlowService:           service.NewFlowService(pgrepo.NewFlowRepository(postgresDB), actionRepo, permissionService),
			HandoverService:       service.NewHandoverService(pgrepo.NewHandoverRepository(postgresDB), actionRepo, permissionService),
			DataAssetService:      service.NewDataAssetService(dataAssetRepo, dataAssetRepo, storageProvider, permissionService),
			CodeRepositoryService: service.NewCodeRepositoryService(codeRepo, codeRepo, permissionService, cfg.CodeRepoRoot, storageProvider),
			AuthService:           authService,
			TokenService:          tokenService,
			AuditService:          auditService,
			MembershipService:     service.NewMembershipService(pgrepo.NewMembershipRepository(postgresDB), permissionService),
		}, nil
	default:
		actionRepo := memory.NewActionRepository()
		authService := service.NewAuthService(memory.NewUserAuthRepository(), tokenService)
		docRepo := memory.NewDocumentRepository()
		versionRepo := memory.NewVersionRepository()
		versionWorkflow := memory.NewVersionWorkflow(versionRepo)
		permissionService := service.NewPermissionService(memory.NewPermissionRepository())
		dataAssetRepo := memory.NewDataAssetRepository()
		codeRepo := memory.NewCodeRepositoryRepository()
		return Container{
			QueueConsumer: publisher,
			QueryService: service.NewQueryService(
				memory.NewTeamSpaceRepository(),
				memory.NewUserQueryRepository(),
				memory.NewProjectRepository(),
			),
			AssistantService:      service.NewAssistantService(publisher, memory.NewAssistantRepository(), docRepo, versionRepo, permissionService),
			DocumentService:       service.NewDocumentService(docRepo, docRepo, storageProvider, versionWorkflow, permissionService),
			AuditQueryService:     service.NewAuditQueryService(memory.NewAuditRepository()),
			DashboardQueryService: service.NewDashboardQueryService(memory.NewDashboardRepository()),
			VersionService:        service.NewVersionService(storageProvider, versionWorkflow, versionRepo, permissionService),
			FlowService:           service.NewFlowService(memory.NewFlowRepository(), actionRepo, permissionService),
			HandoverService:       service.NewHandoverService(memory.NewHandoverRepository(), actionRepo, permissionService),
			DataAssetService:      service.NewDataAssetService(dataAssetRepo, dataAssetRepo, storageProvider, permissionService),
			CodeRepositoryService: service.NewCodeRepositoryService(codeRepo, codeRepo, permissionService, cfg.CodeRepoRoot, storageProvider),
			AuthService:           authService,
			TokenService:          tokenService,
			AuditService:          auditService,
			MembershipService:     service.NewMembershipService(memory.NewMembershipRepository(), permissionService),
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
			Host:               cfg.SynologyHost,
			Port:               cfg.SynologyPort,
			HTTPS:              cfg.SynologyHTTPS,
			InsecureSkipVerify: cfg.SynologyInsecureSkipVerify,
			Account:            cfg.SynologyAccount,
			Password:           cfg.SynologyPassword,
			SharePath:          cfg.SynologySharePath,
		})
	default:
		log.Println("[bootstrap] using memory storage")
		return memstorage.NewProvider()
	}
}
