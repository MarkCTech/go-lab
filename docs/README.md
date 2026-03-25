# Documentation

**Canonical:** HTTP contract → [openapi.yaml](openapi.yaml); Phase A RBAC/routes → [platform-control-plane.md](platform-control-plane.md); migrations/readiness → [migrations.md](migrations.md); roadmap → [MASTER_PLAN.md](MASTER_PLAN.md) (**§7** shipped, **§9** backlog).

**Also:** [CHAT_TODOS.md](../CHAT_TODOS.md) (short session notes). **Integrators:** [platform-api-consumer-brief.md](platform-api-consumer-brief.md) + OpenAPI. Optional Marble/TaskStack onboarding packs: add a folder under `docs/` when you have material and link it here (no reserved path until then).

**Phase 5 (platform):** Desktop handoff is `POST /api/v1/auth/desktop/start` → `POST /api/v1/auth/desktop/exchange` → `POST /api/v1/auth/join-token` (see [desktop-auth-bridge.md](desktop-auth-bridge.md), [openapi.yaml](openapi.yaml), migration `000004_*`).

| Doc | Topics |
|-----|--------|
| [MASTER_PLAN.md](MASTER_PLAN.md) | Roadmap, decisions, shipped, backlog |
| [platform-api-consumer-brief.md](platform-api-consumer-brief.md) | Integration overview and OpenAPI index for TaskStack / Marble |
| [data-ownership.md](data-ownership.md) | Platform vs TaskStack vs Marble; sync / DB performance design |
| [ci.md](ci.md) | GitHub Actions workflow, local checks, OpenAPI validation, testing scope |
| [security-posture.md](security-posture.md) | Security architecture direction; what CI does not replace |
| [auth-session.md](auth-session.md) | Sessions, CSRF, limits, Redis |
| [oidc-auth0.md](oidc-auth0.md) | OIDC Bearer, identities, M2M |
| [adr-account-linking.md](adr-account-linking.md) | Account linking policy |
| [jwt-rotation.md](jwt-rotation.md) | HS256 rotation |
| [bootstrap-sunset.md](bootstrap-sunset.md) | Disabling bootstrap |
| [desktop-auth-bridge.md](desktop-auth-bridge.md) | Desktop handoff (exchange + PKCE + join-token) |
| [platform-admin-ui.md](platform-admin-ui.md) | Admin SPA |
| [platform-control-plane.md](platform-control-plane.md) | Phase A: domain boundaries, RBAC matrix, route ↔ permission map |
| [platform-operator-roles.md](platform-operator-roles.md) | Phase A: SQL to grant `user_platform_roles` |
| [openapi.yaml](openapi.yaml) | Public API contract (OpenAPI 3) |
| [tls-reverse-proxy.md](tls-reverse-proxy.md) | HTTPS, `Secure` cookie, HSTS |
| [ops-secret-rotation.md](ops-secret-rotation.md) | Secret / key rotation checklist |
| [migrations.md](migrations.md) | Migrations, `/readyz`, schema golden |

[Repo README](../README.md) · [api README](../api/README.md) · [Scripts index](../scripts/README.md)
