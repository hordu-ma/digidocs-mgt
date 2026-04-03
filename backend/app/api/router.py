from fastapi import APIRouter

from app.api.routes import assistant, audit_events, auth, dashboard, documents, flows, handovers
from app.api.routes import projects, team_spaces, versions


api_router = APIRouter()
api_router.include_router(auth.router, prefix="/auth", tags=["auth"])
api_router.include_router(team_spaces.router, prefix="/team-spaces", tags=["team-spaces"])
api_router.include_router(projects.router, prefix="/projects", tags=["projects"])
api_router.include_router(documents.router, prefix="/documents", tags=["documents"])
api_router.include_router(versions.router, tags=["versions"])
api_router.include_router(flows.router, prefix="/documents", tags=["flows"])
api_router.include_router(handovers.router, prefix="/handovers", tags=["handovers"])
api_router.include_router(dashboard.router, prefix="/dashboard", tags=["dashboard"])
api_router.include_router(audit_events.router, prefix="/audit-events", tags=["audit-events"])
api_router.include_router(assistant.router, prefix="/assistant", tags=["assistant"])
