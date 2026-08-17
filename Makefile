.PHONY: doctor verify verify-go coverage-go verify-worker coverage-worker verify-frontend coverage-frontend integration-postgres check-doc-sync install-project-skills install-hooks install-persistent-routing smoke smoke-e2e smoke-stack-up smoke-stack-down smoke-stack-status status clean-demo-data clean-demo-data-apply

doctor:
	./scripts/codex/doctor.sh

check-doc-sync:
	./scripts/codex/check-doc-sync.sh

status:
	./scripts/codex/report.sh

# 烟测默认打隔离栈（127.0.0.1:18083），不再碰演示环境。
# 打非烟测环境会被脚本拒绝，除非显式 ALLOW_SMOKE_ON_SHARED=1。
smoke:
	./scripts/codex/smoke-local.sh

# 一条命令跑完整流程：起隔离栈 → 灌种子 → 跑烟测 → 销毁。
smoke-e2e:
	./scripts/codex/smoke-e2e.sh

# 需要保留栈反复调试时，手动管理生命周期。
smoke-stack-up:
	./scripts/codex/smoke-stack.sh up

smoke-stack-down:
	./scripts/codex/smoke-stack.sh down

smoke-stack-status:
	./scripts/codex/smoke-stack.sh status

# 兜底清理：万一有人用 ALLOW_SMOKE_ON_SHARED=1 对着演示库跑过烟测。
clean-demo-data:
	./scripts/codex/clean-demo-data.sh --dry-run

clean-demo-data-apply:
	./scripts/codex/clean-demo-data.sh --apply

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
