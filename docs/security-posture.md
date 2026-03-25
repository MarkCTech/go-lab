# Security posture and roadmap (v0.1)

**Audience:** operators and architects selling or hardening the suite. **Companion:** [ci.md](ci.md) (what automated tests do *not* cover).

## Architecture direction (makes stronger security easier)

1. **Clear trust boundaries:** Browser / desktop / M2M each have documented auth paths ([platform-api-consumer-brief.md](platform-api-consumer-brief.md), [desktop-auth-bridge.md](desktop-auth-bridge.md)). Prefer **no secrets in browser bundles**; server-side holds `PLATFORM_CLIENT_SECRET`.
2. **Defense in depth:** TLS at the edge, tight CORS, HttpOnly cookies + CSRF for session mutations, rate limits + lockout, optional **Redis** for shared limits across replicas ([auth-session.md](auth-session.md)).
3. **Least privilege:** Human-only routes reject `client:*` ([openapi.yaml](openapi.yaml)); scope M2M over time ([MASTER_PLAN.md](MASTER_PLAN.md) §9).
4. **Auditability:** Auth audit events exist; taxonomy can mature ([MASTER_PLAN.md](MASTER_PLAN.md) §9).
5. **Migrations-only schema:** No runtime DDL surprises ([migrations.md](migrations.md)).

None of this replaces a **threat model** or **penetration test**; it makes outcomes more predictable when you add them.

## What to add as the product matures

| Practice | Purpose |
|----------|---------|
| Written **threat model** (STRIDE or similar) | Focus testing on realistic abuse |
| **Dependency scanning** (Dependabot; **`govulncheck`** in CI + `ci-local`) | Known vulnerable dependencies |
| **Fuzzing** (Go fuzz targets on parsers, token paths) | Memory / panic / unexpected inputs |
| **OWASP ASVS** self-assessment (even informally) | Structured hardening checklist |
| **Periodic pen test** or bug bounty | Adversarial validation |
| **OpenAPI ↔ route drift** check | Contract integrity |
| **SAST/DAST** (commercial or OSS) | Broader coverage when budget allows |

## Honest stance on current CI

Unit tests, OpenAPI structural validation, Compose smoke, and schema golden catch **regressions** and **footguns**, not clever abuse. Assume gaps until you run focused security work.

## Related

[ci.md](ci.md) · [tls-reverse-proxy.md](tls-reverse-proxy.md) · [ops-secret-rotation.md](ops-secret-rotation.md) · [MASTER_PLAN.md](MASTER_PLAN.md)
