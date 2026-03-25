# Documentation

**Planning:** [MASTER_PLAN.md](MASTER_PLAN.md) (backlog **§9**, shipped **§7**). **Working notes:** [CHAT_TODOS.md](../CHAT_TODOS.md) — brief session handoff and “next focus” after merges. **External integrators (TaskStack / Marble):** [platform-api-consumer-brief.md](platform-api-consumer-brief.md) + [openapi.yaml](openapi.yaml). **Suite onboarding packs** (Marble/TaskStack): maintain under `docs/suite-onboarding/` when used, and link from this index.

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
| [openapi.yaml](openapi.yaml) | Public API contract (OpenAPI 3) |
| [tls-reverse-proxy.md](tls-reverse-proxy.md) | HTTPS, `Secure` cookie, HSTS |
| [ops-secret-rotation.md](ops-secret-rotation.md) | Secret / key rotation checklist |
| [migrations.md](migrations.md) | Migrations, `/readyz`, schema golden |

[Repo README](../README.md) · [api README](../api/README.md) · [Scripts index](../scripts/README.md)
