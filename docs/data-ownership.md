# Data ownership — Platform, TaskStack, Marble

**Audience:** suite architecture and schema planning. **Go-lab** migrations only cover the **platform** slice; TaskStack and Marble carry their own databases (or schemas) in their repos, linked by **`platform_user_id`** (or equivalent UUID) as a contract.

**Related:** [MASTER_PLAN.md](MASTER_PLAN.md) §2–§6 · [platform-control-plane.md](platform-control-plane.md) (Phase A API/DB slice vs stubs) · [adr-account-linking.md](adr-account-linking.md) · [desktop-auth-bridge.md](desktop-auth-bridge.md) · [openapi.yaml](openapi.yaml)

---

## 1. Three pillars (who owns what)

| Pillar | Owns (examples) | Does **not** own |
|--------|-------------------|------------------|
| **Platform (go-lab)** | Login identifiers (`users`, `user_identities`), sessions, auth audit, future **entitlements** / tenancy keys tied to `users.id` | Game characters, posts, DMs, sim state, rendering |
| **TaskStack** | Website UX data: posts, threads, notifications, workspace membership, settings that are **product-specific** | Opaque session secret state (platform); authoritative gameplay |
| **Marble** | **Authoritative** gameplay: characters, stats, progression, inventory, in-session / in-world mail, match snapshots the sim agrees on | Platform password hashes; TaskStack social graph (unless you deliberately unify) |

**Rule of thumb:** If cheating or desync between players would matter, the **canonical** value for that field should eventually live where **Marble** (or the **host-authoritative** process) can enforce it — not only inside a thin client and not as unverified writes on the platform API.

**Cross-cutting identity:** One **human** should map to one **platform user** (with linking rules in [adr-account-linking.md](adr-account-linking.md)). TaskStack and Marble rows reference that id; they do not redefine “who is logged in” separately without a documented sync story.

---

## 2. TaskStack ↔ platform signup (intended flow)

- **Browser signup in TaskStack:** TaskStack **backend** creates or links a row on the platform (e.g. `POST /api/v1/auth/register` or a future server-only variant). The **browser** never holds `PLATFORM_CLIENT_SECRET`.
- **With vs without session on every call:** The **first** signup/login request is usually **user-driven** (form POST → BFF → platform). **Later** jobs (GDPR export, batch reconcile) may use **scoped** machine credentials when you add them — not the same as giving generic M2M **full** user mutation powers.
- **Native client + browser, same person:** Same **platform** account (email/OIDC linking). Desktop flows may use [desktop-auth-bridge.md](desktop-auth-bridge.md) patterns; web uses cookie or OIDC Bearer per product choice.

---

## 3. Gameplay data: Marble authority, sync, and anti-cheat (design)

You can design this **before** the sim ships.

### Authority model

- **Online multiplayer (fair):** Treat **Marble (dedicated server or elected host)** as the **authority** for mutable gameplay state. Clients send **inputs** or **commands**; the server **validates** and **updates** state. **Never** trust the client as the sole source of truth for stats, currency, or unlocks.
- **Offline / local-only:** Client or local host may own state **until** you require cloud sync or competitive integrity — then define a **migration** path (e.g. “first online bind attaches to platform user”).
- **Platform API:** Stays **out** of per-frame gameplay. It may issue **join tokens**, **entitlements**, or **ban flags** consumed by Marble — not arbitrary “set player level” from an unverified client.

### Sync and performance (DB is not the game loop)

- **Hot path:** Keep **simulation state in memory** on the authoritative process. Target **tick / step** latency, not SQL round-trips per frame.
- **Persistence:** **Batch** writes — checkpoint, round end, inventory change commits, or debounced async **queues** — not every position update.
- **Async pipelines:** Use **outbox / job queue** patterns for non-critical paths (analytics, long-form replays, cross-service fan-out). **Redis** (optional in go-lab today) or a dedicated queue service fits; fail-open vs fail-closed depends on the feature ([auth-session.md](auth-session.md) Redis notes are analogous).
- **Read scaling:** Caches or read replicas for **leaderboards / profiles** are fine if **writes** still go through authority + clear invalidation rules.

### “Mutual source of truth”

- **One logical user** → platform `users.id`.  
- **Gameplay truth** → Marble’s store, keyed by `platform_user_id` + game-specific ids.  
- **“Mutual”** means **consistent references and events**, not one physical MySQL with every subsystem in one table.

---

## 4. Human-only API routes vs “intercepting a key”

Platform **human-only** enforcement (e.g. **`PUT`/`DELETE` `/api/v1/users/:id`** rejects `client:*` subjects) limits what **machine credentials** can do. It does **not** replace:

- **User session theft:** XSS, malware, or physical access can abuse a **real** `user:` session or Bearer token — those requests **pass** human-only checks.
- **Phishing / credential stuffing:** Attacks the **human** path, not M2M.
- **JWT signing key compromise:** Attacker could mint `user:` tokens if **`JWT_SECRET`** leaks — rotation and secrecy are critical ([jwt-rotation.md](jwt-rotation.md)).
- **Insider / DB access:** Out of band of HTTP auth.

So: human-only is **blast-radius control** for **service accounts**, not a guarantee against “intercepting” user keys.

---

## 5. Self-host operator model

- **God-mode on own box:** Operator uses admin UI + human session for dangerous operations; M2M stays limited by design.
- **Future “self-host cloud”:** Same boundaries; add deployment isolation and optional **scoped** service principals per tenant when needed.

---

## 6. Example entity placement (illustrative)

| Entity | Likely owner | Notes |
|--------|--------------|--------|
| `users`, `user_identities`, `auth_sessions` | Platform | go-lab migrations |
| `posts`, `direct_messages` (website) | TaskStack | `platform_user_id` FK |
| `characters`, `player_progress`, `inventory` | Marble | authoritative sim + persistence |
| `auth_audit_events` | Platform | security audit |
| `entitlements` (SKU / feature flags) | Platform (v1 sketch) | consumed by TaskStack/Marble |

Adjust names when each product’s schema is implemented; keep **ownership** stable.
