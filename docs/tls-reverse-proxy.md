# TLS and reverse proxy (production-style)

The Go API listens on **plain HTTP** inside the container. For production, terminate **TLS** at a reverse proxy (Caddy, Nginx, Envoy, cloud load balancer) and forward HTTP to the backend.

## Cookie sessions behind HTTPS

Set **`SESSION_COOKIE_SECURE=true`** in the backend environment when clients only reach the API over HTTPS. Browsers will not send `Secure` cookies over `http://`, which breaks login if the site is mistakenly served without TLS.

Align **`SESSION_SAMESITE`** with your deployment: `Lax` is typical for same-site SPAs; stricter modes may require same-origin API routing.

See [auth-session.md](auth-session.md) and [`.env.example`](../.env.example).

## HSTS (HTTP Strict Transport Security)

After TLS is correctly configured end-to-end, the edge can send:

`Strict-Transport-Security: max-age=31536000; includeSubDomains`

**Caution:** HSTS is sticky. If you enable it before TLS is stable, clients may refuse plain HTTP for `max-age`. Use a short `max-age` during rollout, or serve only over HTTPS from day one on that hostname.

## Trusted headers

If the proxy terminates TLS, configure it to pass the original scheme and host (for example `X-Forwarded-Proto`, `X-Forwarded-Host`, `X-Forwarded-For`) and ensure the backend or proxy sets **`Trust-Forwarded`** semantics appropriate to your stack. Gin’s default client IP uses `RemoteAddr`; behind a proxy you may need trusted proxy settings so rate limits and audit IPs stay accurate.

## Example (conceptual)

- **Caddy:** automatic HTTPS; `reverse_proxy backend:5000`.
- **Nginx:** `listen 443 ssl`; `proxy_pass http://backend:5000`; set `proxy_set_header X-Forwarded-Proto $scheme` (and related headers).

Keep **`CORS_ALLOWED_ORIGINS`** on HTTPS origins only when the SPA is served over HTTPS.

## Related

- [auth-session.md](auth-session.md) — CSRF, cookies, CORS.
- [openapi.yaml](openapi.yaml) — API contract.
