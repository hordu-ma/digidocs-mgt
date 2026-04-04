.PHONY: doctor verify verify-go verify-worker verify-frontend check-doc-sync install-project-skills install-hooks smoke status

doctor:
	./scripts/codex/doctor.sh

check-doc-sync:
	./scripts/codex/check-doc-sync.sh

status:
	./scripts/codex/report.sh

smoke:
	./scripts/codex/smoke-local.sh

verify: doctor check-doc-sync verify-go verify-worker verify-frontend

verify-go:
	cd backend-go && go test ./...

verify-worker:
	cd backend-py-worker && .venv/bin/python -m pytest -q

verify-frontend:
	cd frontend && npm run build

install-project-skills:
	./scripts/codex/install-project-skills.sh

install-hooks:
	./scripts/codex/install-hooks.sh
