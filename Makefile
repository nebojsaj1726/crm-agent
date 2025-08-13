# ----------- Configuration -----------
include .env
export

OLLAMA_MODEL ?= $(shell echo $$OLLAMA_MODEL)
EMBED_MODEL ?= $(shell echo $$EMBED_MODEL)

# ----------- Main commands -----------

.PHONY: start stop pull-models build wait-chroma seed check-binary query web logs setup clean

start:
	docker-compose up -d

pull-models:
	docker exec ollama ollama pull $(OLLAMA_MODEL)
	docker exec ollama ollama pull $(EMBED_MODEL)

build:
	go build -o crm

wait-chroma:
	@echo "Waiting for Chroma to be ready..."
	@until curl -s http://localhost:8000/api/v1/heartbeat > /dev/null; do \
		echo "Waiting for Chroma..."; sleep 2; \
	done
	@echo "Chroma is ready!"

seed: wait-chroma build
	./crm -seed

check-binary:
	@if [ ! -f ./crm ]; then \
		echo "Error: crm binary not found. Run 'make setup' first"; \
		exit 1; \
	fi

query: check-binary
	./crm -query

web: check-binary
	@if lsof -i:8080 > /dev/null; then \
		echo "API server already running on port 8080"; \
	else \
		./crm -web & \
	fi

	cd web && [ -d node_modules ] || npm install
	cd web && npm run dev

logs:
	docker-compose logs -f


setup: start pull-models seed
	@echo "✔ Setup complete. Run 'make query' to use the app or 'make logs' to see container output."

clean:
	-docker stop ollama || true
	-docker rm ollama || true
	-docker stop chroma || true
	-docker rm chroma || true
	-docker-compose down
	-rm -f crm
