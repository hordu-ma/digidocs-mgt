.PHONY: doctor verify verify-go coverage-go verify-worker coverage-worker verify-frontend coverage-frontend check-doc-sync install-project-skills install-hooks install-persistent-routing smoke status

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

coverage-go:
	cd backend-go && go test ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic
	cd backend-go && go tool cover -func=coverage.out | tail -n 1

verify-worker:
	cd backend-py-worker && uv run pytest -q

coverage-worker:
	cd backend-py-worker && uv run pytest --cov=app --cov-report=term-missing -q

verify-frontend:
	cd frontend && npm run test:run
	cd frontend && npm run build

coverage-frontend:
	cd frontend && npm run test:coverage

install-project-skills:
	./scripts/codex/install-project-skills.sh

install-hooks:
	./scripts/codex/install-hooks.sh

install-persistent-routing:
	./scripts/codex/install-persistent-routing.sh
