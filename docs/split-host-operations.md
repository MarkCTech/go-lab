# Split-host operations checklist

Use this when platform API, TaskStack, and Marble run on different hosts.

## Platform capabilities covered here

- Restore governance workflow (`/api/v1/backups/*`) with two-person approval logic.
- Readiness payload support for migration gating via `/readyz` when `MIGRATION_EXPECTED_VERSION` is set.
- Structured logs for restore workflow events.

Platform API does not execute physical backup/restore operations.

## Deployment checklist

| Area | Requirement |
|------|-------------|
| TLS | Terminate at edge; enforce secure cookie settings |
| CORS | Explicit browser origins only; no wildcard in production |
| Proxy trust | Configure trusted proxy/IP handling for accurate client IP and rate limiting |
| Readiness | Gate rollout on `/readyz` and expected migration version |
| Secrets | Rotate JWT/client/DB/OIDC secrets per runbook |

## Suite integration checklist

- TaskStack server-side integration for sensitive platform operations.
- Desktop exchange -> bearer -> join-token consumer flow implemented in suite clients.
- Marble validates join token claims (`token_use=join`) and enforces game-side trust.
- Cross-repo observability correlation (`request_id` propagation) where possible.

## Related

- [platform-control-plane.md](platform-control-plane.md)
- [platform-api-consumer-brief.md](platform-api-consumer-brief.md)
- [desktop-auth-bridge.md](desktop-auth-bridge.md)
- [tls-reverse-proxy.md](tls-reverse-proxy.md)
- [ops-secret-rotation.md](ops-secret-rotation.md)
