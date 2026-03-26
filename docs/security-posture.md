# Security posture

This repo has practical baseline controls, not a complete security assurance program.

## Current baseline

- Cookie sessions + CSRF for browser mutations.
- Optional OIDC bearer validation with `(issuer, sub)` identity mapping.
- Rate limits and email lockout (optional shared state via Redis).
- Human-only enforcement on selected privileged routes.
- Immutable auth/admin audit tables.
- Migration-only schema changes.

## What this enables

- Predictable trust boundaries across browser, desktop, and machine clients.
- Better blast-radius control for machine credentials.
- Auditable privileged workflow actions.

## What is still needed for stronger assurance

- Threat modeling and abuse-case review.
- Fuzzing for parser/token and boundary handlers.
- Periodic pentest or adversarial assessment.
- Optional OpenAPI-to-route drift enforcement.
- Performance/load validation for production-like traffic.

## CI reality

CI validates correctness and drift checks, not adversarial resilience.

## Related

- [ci.md](ci.md)
- [auth-session.md](auth-session.md)
- [tls-reverse-proxy.md](tls-reverse-proxy.md)
- [ops-secret-rotation.md](ops-secret-rotation.md)
