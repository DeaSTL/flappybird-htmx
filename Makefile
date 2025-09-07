.PHONY: dev prod build-dev build-prod clean rebuild rebuild-prod fresh fresh-dev

# Development with hot reload (logging middleware enabled)
dev:
	DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml up --build

# Development build only
build-dev:
	DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml build

# Production build and run (logging middleware disabled)
prod:
	DOCKER_BUILDKIT=1 docker compose up --build

# Production build only
build-prod:
	DOCKER_BUILDKIT=1 docker compose build

# Force rebuild production without cache (use when code changes aren't picked up)
rebuild-prod:
	DOCKER_BUILDKIT=1 docker compose build --no-cache
	DOCKER_BUILDKIT=1 docker compose up

# Force rebuild development without cache
rebuild-dev:
	DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml build --no-cache
	DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml up

# Complete fresh start - removes containers and images, then rebuilds
fresh:
	docker compose down --rmi all --volumes --remove-orphans
	DOCKER_BUILDKIT=1 docker compose up --build

# Complete fresh start for development
fresh-dev:
	docker compose -f docker-compose.dev.yml down --rmi all --volumes --remove-orphans
	DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml up --build

# Clean up containers and caches
clean:
	docker compose down
	docker compose -f docker-compose.dev.yml down
	docker volume rm flappybird-htmx_go-modules flappybird-htmx_go-build-cache 2>/dev/null || true

# Run locally without Docker (requires Go installed)
local:
	cd app && go run .

# Time the build
time-build:
	time DOCKER_BUILDKIT=1 docker compose build --no-cache

time-build-dev:
	time DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml build --no-cache