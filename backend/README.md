# Backend

## Setup

```bash
uv sync
uv run uvicorn app.main:app --reload
```

## Migrate

```bash
uv run alembic upgrade head
```

