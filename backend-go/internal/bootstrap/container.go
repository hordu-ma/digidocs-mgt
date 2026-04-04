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
	DB                   *sql.DB
	QueryService         service.QueryService
	VersionQueryService  service.VersionQueryService
	FlowQueryService     service.FlowQueryService
	HandoverQueryService service.HandoverQueryService
	TaskService          service.TaskService
	ActionService        service.ActionService
	TokenService         service.TokenService
	AuditService         service.AuditService
	UploadService        service.UploadService
}

func BuildContainer(cfg config.Config) (Container, error) {
	publisher := memqueue.NewPublisher()
	storageProvider := memstorage.NewProvider()
	tokenService := service.NewTokenService(cfg.JWTSecret)
	auditService := service.NewAuditService()
	uploadService := service.NewUploadService(storageProvider)
	actionService := service.NewActionService()

	switch cfg.DataBackend {
	case "postgres":
		postgresDB, err := db.OpenPostgres(cfg.DatabaseURL)
		if err != nil {
			return Container{}, err
		}

		return Container{
			DB: postgresDB,
			QueryService: service.NewQueryService(
				pgrepo.NewTeamSpaceRepository(postgresDB),
				pgrepo.NewProjectRepository(postgresDB),
				pgrepo.NewDocumentRepository(postgresDB),
			),
			VersionQueryService:  service.NewVersionQueryService(memory.NewVersionRepository()),
			FlowQueryService:     service.NewFlowQueryService(memory.NewFlowRepository()),
			HandoverQueryService: service.NewHandoverQueryService(memory.NewHandoverRepository()),
			TaskService:          service.NewTaskService(publisher),
			ActionService:        actionService,
			TokenService:         tokenService,
			AuditService:         auditService,
			UploadService:        uploadService,
		}, nil
	case "memory":
		return Container{
			QueryService: service.NewQueryService(
				memory.NewTeamSpaceRepository(),
				memory.NewProjectRepository(),
				memory.NewDocumentRepository(),
			),
			VersionQueryService:  service.NewVersionQueryService(memory.NewVersionRepository()),
			FlowQueryService:     service.NewFlowQueryService(memory.NewFlowRepository()),
			HandoverQueryService: service.NewHandoverQueryService(memory.NewHandoverRepository()),
			TaskService:          service.NewTaskService(publisher),
			ActionService:        actionService,
			TokenService:         tokenService,
			AuditService:         auditService,
			UploadService:        uploadService,
		}, nil
	default:
		return Container{
			QueryService: service.NewQueryService(
				memory.NewTeamSpaceRepository(),
				memory.NewProjectRepository(),
				memory.NewDocumentRepository(),
			),
			VersionQueryService:  service.NewVersionQueryService(memory.NewVersionRepository()),
			FlowQueryService:     service.NewFlowQueryService(memory.NewFlowRepository()),
			HandoverQueryService: service.NewHandoverQueryService(memory.NewHandoverRepository()),
			TaskService:          service.NewTaskService(publisher),
			ActionService:        actionService,
			TokenService:         tokenService,
			AuditService:         auditService,
			UploadService:        uploadService,
		}, nil
	}
}
