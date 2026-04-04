package bootstrap

import (
	"database/sql"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/db"
	memqueue "digidocs-mgt/backend-go/internal/queue/memory"
	"digidocs-mgt/backend-go/internal/repository/memory"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
	"digidocs-mgt/backend-go/internal/service"
	memstorage "digidocs-mgt/backend-go/internal/storage/memory"
)

type Container struct {
	DB                     *sql.DB
	QueryService           service.QueryService
	AuditQueryService      service.AuditQueryService
	DashboardQueryService  service.DashboardQueryService
	VersionQueryService    service.VersionQueryService
	VersionCommandService  service.VersionCommandService
	VersionWorkflowService service.VersionWorkflowService
	FlowQueryService       service.FlowQueryService
	HandoverQueryService   service.HandoverQueryService
	TaskService            service.TaskService
	ActionService          service.ActionService
	TokenService           service.TokenService
	AuditService           service.AuditService
	UploadService          service.UploadService
}

func BuildContainer(cfg config.Config) (Container, error) {
	publisher := memqueue.NewPublisher()
	storageProvider := memstorage.NewProvider()
	tokenService := service.NewTokenService(cfg.JWTSecret)
	auditService := service.NewAuditService()
	uploadService := service.NewUploadService(storageProvider)

	switch cfg.DataBackend {
	case "postgres":
		postgresDB, err := db.OpenPostgres(cfg.DatabaseURL)
		if err != nil {
			return Container{}, err
		}
		actionService := service.NewActionService(pgrepo.NewActionRepository(postgresDB))

		return Container{
			DB: postgresDB,
			QueryService: service.NewQueryService(
				pgrepo.NewTeamSpaceRepository(postgresDB),
				pgrepo.NewProjectRepository(postgresDB),
				pgrepo.NewDocumentRepository(postgresDB),
			),
			AuditQueryService:      service.NewAuditQueryService(pgrepo.NewAuditRepository(postgresDB)),
			DashboardQueryService:  service.NewDashboardQueryService(pgrepo.NewDashboardRepository(postgresDB)),
			VersionQueryService:    service.NewVersionQueryService(pgrepo.NewVersionRepository(postgresDB)),
			VersionCommandService:  service.NewVersionCommandService(pgrepo.NewVersionRepository(postgresDB)),
			VersionWorkflowService: service.NewVersionWorkflowService(pgrepo.NewVersionWorkflow(postgresDB)),
			FlowQueryService:       service.NewFlowQueryService(pgrepo.NewFlowRepository(postgresDB)),
			HandoverQueryService:   service.NewHandoverQueryService(pgrepo.NewHandoverRepository(postgresDB)),
			TaskService:            service.NewTaskService(publisher),
			ActionService:          actionService,
			TokenService:           tokenService,
			AuditService:           auditService,
			UploadService:          uploadService,
		}, nil
	case "memory":
		actionService := service.NewActionService(memory.NewActionRepository())
		return Container{
			QueryService: service.NewQueryService(
				memory.NewTeamSpaceRepository(),
				memory.NewProjectRepository(),
				memory.NewDocumentRepository(),
			),
			AuditQueryService:      service.NewAuditQueryService(memory.NewAuditRepository()),
			DashboardQueryService:  service.NewDashboardQueryService(memory.NewDashboardRepository()),
			VersionQueryService:    service.NewVersionQueryService(memory.NewVersionRepository()),
			VersionCommandService:  service.NewVersionCommandService(memory.NewVersionRepository()),
			VersionWorkflowService: service.NewVersionWorkflowService(memory.NewVersionWorkflow()),
			FlowQueryService:       service.NewFlowQueryService(memory.NewFlowRepository()),
			HandoverQueryService:   service.NewHandoverQueryService(memory.NewHandoverRepository()),
			TaskService:            service.NewTaskService(publisher),
			ActionService:          actionService,
			TokenService:           tokenService,
			AuditService:           auditService,
			UploadService:          uploadService,
		}, nil
	default:
		actionService := service.NewActionService(memory.NewActionRepository())
		return Container{
			QueryService: service.NewQueryService(
				memory.NewTeamSpaceRepository(),
				memory.NewProjectRepository(),
				memory.NewDocumentRepository(),
			),
			AuditQueryService:      service.NewAuditQueryService(memory.NewAuditRepository()),
			DashboardQueryService:  service.NewDashboardQueryService(memory.NewDashboardRepository()),
			VersionQueryService:    service.NewVersionQueryService(memory.NewVersionRepository()),
			VersionCommandService:  service.NewVersionCommandService(memory.NewVersionRepository()),
			VersionWorkflowService: service.NewVersionWorkflowService(memory.NewVersionWorkflow()),
			FlowQueryService:       service.NewFlowQueryService(memory.NewFlowRepository()),
			HandoverQueryService:   service.NewHandoverQueryService(memory.NewHandoverRepository()),
			TaskService:            service.NewTaskService(publisher),
			ActionService:          actionService,
			TokenService:           tokenService,
			AuditService:           auditService,
			UploadService:          uploadService,
		}, nil
	}
}
