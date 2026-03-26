# Documentation

Use this index to find the canonical source for each topic.

## Canonical sources

- API contract: [openapi.yaml](openapi.yaml)
- Product state and priorities: [MASTER_PLAN.md](MASTER_PLAN.md)
- Route permissions and boundaries: [platform-control-plane.md](platform-control-plane.md)
- Schema and readiness rules: [migrations.md](migrations.md)
- Environment variables: [`.env.example`](../.env.example)
- First local run: [install-and-play.md](install-and-play.md)

## Domain briefs

| Doc | Purpose |
|-----|---------|
| [MASTER_PLAN.md](MASTER_PLAN.md) | Snapshot, decisions, shipped work, active backlog |
| [platform-api-consumer-brief.md](platform-api-consumer-brief.md) | Integration contract usage (TaskStack/Marble/other clients) |
| [platform-control-plane.md](platform-control-plane.md) | RBAC boundaries, permissions, privileged workflow rules |
| [platform-admin-ui.md](platform-admin-ui.md) | Admin SPA scope, route mapping, session behavior |
| [platform-operator-roles.md](platform-operator-roles.md) | How operator roles are assigned in SQL |
| [migrations.md](migrations.md) | Migration chain, `/readyz` version gate, schema golden |
| [install-and-play.md](install-and-play.md) | Newcomer quickstart: run stack, verify, local UI preview |
| [auth-session.md](auth-session.md) | Cookie sessions, CSRF, lockout/limits, desktop bridge links |
| [oidc-auth0.md](oidc-auth0.md) | OIDC Bearer validation and identity linking constraints |
| [jwt-rotation.md](jwt-rotation.md) | HS256 JWT signing secret rotation runbook |
| [desktop-auth-bridge.md](desktop-auth-bridge.md) | Desktop exchange + PKCE + join-token flow |
| [data-ownership.md](data-ownership.md) | Platform vs TaskStack vs Marble ownership model |
| [ci.md](ci.md) | CI jobs and local CI-equivalent commands |
| [security-posture.md](security-posture.md) | Security stance and hardening roadmap |
| [tls-reverse-proxy.md](tls-reverse-proxy.md) | HTTPS/cookie/proxy requirements |
| [ops-secret-rotation.md](ops-secret-rotation.md) | Rotation checklist for secrets and credentials |
| [bootstrap-sunset.md](bootstrap-sunset.md) | Bootstrap bridge retirement checklist |
| [adr-account-linking.md](adr-account-linking.md) | Account linking policy ADR |
| [split-host-operations.md](split-host-operations.md) | Split-host runbook and integration checklist |

## Notes and module docs

- Short-lived notes: [CHAT_TODOS.md](../CHAT_TODOS.md) (do not treat as canonical)
- Repo overview: [README.md](../README.md)
- API module: [api/README.md](../api/README.md)
- Scripts: [scripts/README.md](../scripts/README.md)
