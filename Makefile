.PHONY: doctor verify verify-go coverage-go verify-worker coverage-worker verify-frontend coverage-frontend integration-postgres check-doc-sync install-project-skills install-hooks install-persistent-routing smoke smoke-e2e status clean-demo-data

doctor:
	./scripts/codex/doctor.sh

check-doc-sync:
	./scripts/codex/check-doc-sync.sh

status:
	./scripts/codex/report.sh

smoke:
	./scripts/codex/smoke-local.sh

# smoke 会往目标库写入一次性数据（版本 / 会话 / 注册账号 / 邀请链接）。
# 对着演示库跑过之后，用这个把残留清掉再给客户演示。
clean-demo-data:
	./scripts/codex/clean-demo-data.sh --dry-run

clean-demo-data-apply:
	./scripts/codex/clean-demo-data.sh --apply

smoke-e2e:
	./scripts/codex/smoke-e2e.sh

integration-postgres:
	./scripts/codex/postgres-integration.sh

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
