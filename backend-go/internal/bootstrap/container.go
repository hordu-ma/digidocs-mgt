package bootstrap

import (
	"database/sql"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/db"
	"digidocs-mgt/backend-go/internal/queue"
	memqueue "digidocs-mgt/backend-go/internal/queue/memory"
	"digidocs-mgt/backend-go/internal/repository/memory"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
	"digidocs-mgt/backend-go/internal/service"
	memstorage "digidocs-mgt/backend-go/internal/storage/memory"
)

type Container struct {
	DB                    *sql.DB
	QueueConsumer         queue.Consumer
	QueryService          service.QueryService
	AuditQueryService     service.AuditQueryService
	DashboardQueryService service.DashboardQueryService
	VersionService        service.VersionService
	FlowService           service.FlowService
	HandoverService       service.HandoverService
	TaskService           service.TaskService
	AuthService           service.AuthService
	TokenService          service.TokenService
	AuditService          service.AuditService
}

func BuildContainer(cfg config.Config) (Container, error) {
	publisher := memqueue.NewPublisher()
	storageProvider := memstorage.NewProvider()
	tokenService := service.NewTokenService(cfg.JWTSecret)
	auditService := service.NewAuditService()

	switch cfg.DataBackend {
	case "postgres":
		postgresDB, err := db.OpenPostgres(cfg.DatabaseURL)
		if err != nil {
			return Container{}, err
		}
		actionRepo := pgrepo.NewActionRepository(postgresDB)
		authService := service.NewAuthService(pgrepo.NewUserAuthRepository(postgresDB), tokenService)
		versionRepo := pgrepo.NewVersionRepository(postgresDB)

		return Container{
			DB:            postgresDB,
			QueueConsumer: publisher,
			QueryService: service.NewQueryService(
				pgrepo.NewTeamSpaceRepository(postgresDB),
				pgrepo.NewProjectRepository(postgresDB),
				pgrepo.NewDocumentRepository(postgresDB),
			),
			AuditQueryService:     service.NewAuditQueryService(pgrepo.NewAuditRepository(postgresDB)),
			DashboardQueryService: service.NewDashboardQueryService(pgrepo.NewDashboardRepository(postgresDB)),
			VersionService:        service.NewVersionService(storageProvider, pgrepo.NewVersionWorkflow(postgresDB), versionRepo),
			FlowService:           service.NewFlowService(pgrepo.NewFlowRepository(postgresDB), actionRepo),
			HandoverService:       service.NewHandoverService(pgrepo.NewHandoverRepository(postgresDB), actionRepo),
			TaskService:           service.NewTaskService(publisher),
			AuthService:           authService,
			TokenService:          tokenService,
			AuditService:          auditService,
		}, nil
	default:
		actionRepo := memory.NewActionRepository()
		authService := service.NewAuthService(memory.NewUserAuthRepository(), tokenService)
		return Container{
			QueueConsumer: publisher,
			QueryService: service.NewQueryService(
				memory.NewTeamSpaceRepository(),
				memory.NewProjectRepository(),
				memory.NewDocumentRepository(),
			),
			AuditQueryService:     service.NewAuditQueryService(memory.NewAuditRepository()),
			DashboardQueryService: service.NewDashboardQueryService(memory.NewDashboardRepository()),
			VersionService:        service.NewVersionService(storageProvider, memory.NewVersionWorkflow(), memory.NewVersionRepository()),
			FlowService:           service.NewFlowService(memory.NewFlowRepository(), actionRepo),
			HandoverService:       service.NewHandoverService(memory.NewHandoverRepository(), actionRepo),
			TaskService:           service.NewTaskService(publisher),
			AuthService:           authService,
			TokenService:          tokenService,
			AuditService:          auditService,
		}, nil
	}
}
