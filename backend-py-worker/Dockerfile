FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    poppler-utils \
    antiword \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY README.md ./
COPY app ./app

CMD ["python", "-m", "app.main"]
