# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Running the Application
```bash
# Start the application using Docker Compose
docker compose up

# For development with hot reload
docker compose up --watch
```

The application runs on `localhost:3200` by default.

### Database Operations
```bash
# Generate database code from SQL schema and queries (requires sqlc)
cd app && sqlc generate
```

The project uses sqlc to generate type-safe Go code from SQL.

## Architecture Overview

This is a real-time multiplayer Flappy Bird game built with Go, HTMX, and server-sent events for live updates.

### Core Components

- **Main Server** (`app/main.go`): Chi router-based HTTP server that serves the game
- **Game Engine** (`app/game/`): Contains game state management, physics, and player/pipe logic
  - `server-state.go`: Manages multiple game sessions and player connections
  - `game-state.go`: Individual game instance state and logic
  - `physics/collision-box.go`: Collision detection system
  - `player.go` & `pipe-set.go`: Game entity management
- **Database Layer** (`app/db/`): SQLite database with sqlc-generated queries for user management
- **Templates** (`app/templates/`): HTMX-based HTML templates for real-time UI updates
- **Assets** (`app/local/`): Game sprites and graphics

### Key Technical Details

- Uses Chi router with compression middleware
- Session-based player management with JWT tokens
- Real-time updates via HTMX polling and server-sent events
- SQLite database for persistent user scores
- Template-driven rendering for dynamic game states
- Docker containerized with multi-architecture support

### File Structure Notes

- All Go code is in the `app/` directory
- Database schema is in `app/db/schema.sql`
- SQL queries are in `app/db/queries.sql` (auto-generated Go code via sqlc)
- Static assets are served from `app/local/`
- Infrastructure code (Terraform) is in `infra/`
- only run the project with the make scripts or docker compose