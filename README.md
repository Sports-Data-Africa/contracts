# SportsData Africa — Platform Contracts (`github.com/Sports-Data-Africa/contracts`)

**Current Version:** `v1.2.0` (see [`VERSION`](./VERSION))

The `contracts` module is the authoritative shared foundation across all SportsData Africa microservices (`prematch-service`, `live-service`, `auth-service`, `bets-service`, `settlement-service`, `results-service`, `wallet-service`). It enforces canonical identity invariants, market grading rules, B2B payload sanitization, gRPC authentication contracts, and cross-service domain events.

---

## 1. Core Architecture & Identity Layer Separation

SportsData Africa enforces a strict separation between **Fixture Identity**, **League/Competition Identity**, and **Market Identity**:

| Identity Layer | External B2B / Client Identifier | Internal Provider Alias | Authoritative Package |
| :--- | :--- | :--- | :--- |
| **Fixture ID** | SDA Canonical Fixture ID (`SPD-FB-100001` / 8-digit canonical ID) | Preserved in `fixture_aliases` (`provider` + `provider_fixture_id`) for live linking & settlement | `contracts/identity`, `contracts/canonical` |
| **League / Competition ID** | Unified Canonical Tournament ID (e.g. `17` Premier League, `18` La Liga, `7` UCL, `4512` FKF PL, `132` NBA) | Mapped via `provider_competition_mappings` (fail-closed on unverified third-party IDs) | Service `identity/competition_resolver` |
| **Market & Selection ID** | Canonical Market Code (`FB_MATCH_WINNER_1X2`, `FB_TOTAL_GOALS`, etc.) | Mapped via `contracts/market/provider_mapping.go` | `contracts/market` |

### Key Invariants
1. **Fixture Resolver Never Modifies League Identity**: [`identity.Engine.ResolveCanonicalID`](./identity/engine.go) resolves `provider_fixture_id -> CanonicalFixtureID` only. It never invents fixture IDs on a miss (`ErrAliasNotFound`) and never alters competition IDs.
2. **Prefix-Safe Alias Resolution**: [`identity.NormalizeProviderFixtureID`](./identity/engine.go) strips `sr:match:` and `sr:sport_event:` in any order (including repeated prefixes and whitespace), rejects bare prefixes without an underlying ID, and preserves both the original trimmed provider reference and the stripped ID in L1/L2 caches and MySQL.
3. **Database Authority Over Cache**: Cache writes in [`identity.TieredIdentityCache`](./identity/cache.go) occur only after authoritative repository writes succeed, and repository errors are never masked by cache misses.
4. **Zero Provider Leakage**: [`b2b`](./b2b) enforces that external B2B payloads and webhooks never expose internal upstream provider names.

---

## 2. Package Reference

- **`canonical/`** — Canonical fixture ID formatting/validation (`SPD-<SPORT>-<SEQ>`), natural match candidate cleaning (`NaturalCandidate`), team name normalization, and canonical provider code normalization (`NormalizeProvider`).
- **`identity/`** — Grade-A 3-tier fixture identity resolution engine (`Engine`, `L1Cache`, `TieredIdentityCache`, MySQL `Repository`), JIT live event matching (`ResolveJITLiveEvent`), and settleability gating (`EvaluateSettleability`).
- **`market/`** — Multi-sport canonical market catalogue, settlement grading rules (Football, Basketball, Tennis, Ice Hockey, Volleyball, Cricket, Rugby, Table Tennis, MMA, Handball, American Football, Baseball, 1UP/2UP Early Payout, Asian Handicap), and provider market mapping.
- **`auth/` & `auth/v1/`** — Sovereign B2B authentication middleware, API key & HMAC signature verification, and gRPC token/entitlement validation (`auth.proto`).
- **`events/`** — Cross-service event schemas (`ClientEntitlementsChangedEvent`, settlement & result certification events).
- **`b2b/`** — B2B client data transfer models and provider-leakage prevention tests.
- **`errors/`** — Standardized platform error codes and HTTP/gRPC status mappings.

---

## 3. Running Verification & Tests

```bash
cd contracts
go test -v ./...
```

---

## 4. Changelog & Version History

### `v1.2.0` (2026-10-30)
- **`identity/engine.go` & `identity/cache.go`**:
  - Added [`NormalizeProviderFixtureID`](./identity/engine.go) to centralize prefix stripping (`sr:match:`, `sr:sport_event:` in any order, repeated prefixes, whitespace trimming, and empty/bare-prefix rejection) while preserving both original and stripped candidate IDs.
  - Standardized `canonical.NormalizeProvider(provider)` across `ResolveCanonicalID`, `ResolveOrCreateFixture`, `AttachAlias`, `ResolveJITLiveEvent`, `L1Cache`, and `TieredIdentityCache`.
  - Ensured repository errors propagate immediately and never get masked by cache misses or fallback loops.
  - Documented and enforced strict separation between fixture identity resolution and league/competition identity.
- **`go.sum`**:
  - Added complete module checksums for `golang.org/x/net@v0.58.0`, `golang.org/x/sys@v0.48.0`, and `golang.org/x/text@v0.42.0`.

### `v1.1.0` (2026-10-29)
- **`events/events.go`**:
  - Added durable entitlement projection event contracts (`ClientEntitlementsChangedEvent`) supporting `ALL | SELECTED | NONE` league access policies and versioned outbox projection.
- **`market/`**:
  - Expanded multi-sport settlement rules and reconciliation test suites.

### `v1.0.0` (2026-10-07)
- Initial Grade-A canonical identity engine (`SPD-<SPORT>-<SEQ>`), 16-invariant E2E test suite, multi-sport market registry, and gRPC auth validator.

