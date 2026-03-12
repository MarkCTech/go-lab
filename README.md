# go-lab

Go backends (Gin, MySQL) and Angular frontends—consolidated from former standalone repos. You’ll normalize modules/layout later; this preserves each project as its own directory.

## Layout

| Directory | Former repo | Notes |
|-----------|-------------|--------|
| `gin_server/` | gin_server | Gin, embedded static files |
| `go_CRUD_api/` | go_CRUD_api | Gin + MySQL JSON CRUD |
| `sql_setup/` | sql_setup | MySQL connect/add/search in Go |
| `GoAngular/` | GoAngular | Angular build embedded in Go (gin), localhost:5000 |
| `AUsers/` | AUsers | Angular CRUD SPA (Angular CLI) |

Each folder keeps its original README and build instructions.

## Consolidated from

Previous standalone repos can be archived/privatized after this repo is the canonical copy.
