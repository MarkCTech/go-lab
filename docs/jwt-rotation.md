# JWT signing key rotation (HS256)

## Dual-secret window

Set **`JWT_SECRET`** to the **new** primary secret (≥32 characters) and **`JWT_SECRET_PREVIOUS`** to the **old** secret (≥32 characters, or omit/empty when not rotating).

- **Minting:** `TokenService` signs only with `JWT_SECRET`.
- **Verification:** `ParseAccessToken` accepts signatures made with **either** `JWT_SECRET` or `JWT_SECRET_PREVIOUS`.

This allows a rolling restart: deploy new config, wait for old JWTs to expire (≤ `JWT_ACCESS_TTL_SECONDS`), then clear `JWT_SECRET_PREVIOUS` in a second deploy.

## Optional metadata

- **`JWT_ACTIVE_KEY_ID`:** Logged at startup for observability; does not change verification behavior.

## Operational steps

1. Generate a new random `JWT_SECRET` (≥32 bytes).
2. Set `JWT_SECRET_PREVIOUS` to the current `JWT_SECRET`, then set `JWT_SECRET` to the new value.
3. Deploy all API instances with the same pair.
4. After **at least** `JWT_ACCESS_TTL_SECONDS` (plus client clock skew buffer), remove `JWT_SECRET_PREVIOUS` from env and redeploy.

## Limits

- Only **one** previous secret is supported (not a full JWKS).
- For asymmetric signing (RS256) or multiple active kids, plan a separate change (P1+).
