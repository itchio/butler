# Butlerd integration for owned bundles (issue #313)

## Context

itch.io has historically sold very large bundles (Bundle for Racial Justice and Equality, Bundle for Ukraine, etc.) containing thousands of games. To avoid flooding the database with millions of `download_keys` rows, ownership of games inside these bundles is **not materialized as `DownloadKey` rows at purchase time**. Instead the user's account owns a `BundleDownloadKey` per bundle purchase, and an individual `DownloadKey` is created lazily on first access of a specific game.

This currently breaks the desktop app: a user who owns thousands of games via a bundle but hasn't installed any of them yet sees "Buy now" on every game page, because `getGameStatus` only treats a game as owned when a `DownloadKey` is present.

The fix has two halves:
1. **Ownership inference**: butler must know a profile owns a game via a bundle, even when no `DownloadKey` exists locally.
2. **Lazy materialization**: when the user actually installs such a game, butler asks the server to mint a real `DownloadKey` and from then on treats the game like any other directly-owned game.

This plan covers butler-side work only. The go-itchio version vendored in `go.mod` already exposes the bundle types/endpoints and the `OwnerID` field on `BundleKey`. Renderer changes are out of scope and will follow in a separate issue.

## Approach

Mirror the existing **Collection** plumbing for browsing bundle contents, since `Bundle`/`BundleGame` are structurally identical to `Collection`/`CollectionGame`. Mirror **OwnedKey** plumbing for bundle ownership, since `BundleKey` is the bundle analogue of `DownloadKey` and must be attached to a profile through `owner_id`.

Add a profile-wide local bundle ownership sync, targeted local ownership checks, a single butler-local hook on `GameAccess` to remember "owned via bundle, not yet claimed", and a single materialization gate at install time.

No separate single-bundle fetch endpoint: users only encounter bundles they own, so `FetchProfileOwnedBundles` already syncs the full `Bundle` metadata (embedded inside each `BundleKey`) into the local DB; the bundle detail page reads its header from cache and the game grid is driven by paginated `FetchBundleGames`.

## Product semantics

Bundles are separate library entities, not an expansion of the existing owned-games list.

- `Fetch.ProfileOwnedKeys` and the existing "Owned" library page continue to mean "games with materialized `DownloadKey` rows." Bundle-contained games do **not** appear there until `ClaimBundleGame` creates a real download key.
- Bundle ownership affects direct game-page status: if a user navigates to a game that exists in an owned bundle, the CTA should behave as owned/installable even before materialization.
- Bundle contents are browsed through a separate bundle library surface: owned bundles list, then a bundle detail page that behaves like a pseudo-collection view.
- Installing a bundle-owned game materializes that specific game into a `DownloadKey`; after that, it appears in the normal owned-key flows like any other directly-owned game.
- This preserves the core goal: large bundles do not flood the main owned library up front, while still allowing users to discover bundle contents from bundle pages and install direct game visits without seeing "Buy now."

## Observed API shape and sync implications

Example `/profile/owned-bundles` output for a standard long-time account contains 15 `bundle_keys`, with bundle sizes ranging from 2 games to 1,741 games. The total membership count for that sample is 4,458 bundle-game rows.

Important observations:

- The owned-bundle list itself is small. The current go-itchio endpoint is not paginated and is documented as capped at 100 bundle keys. Butler should fetch and persist the whole owned-bundle key list at once, then page/sort/search locally for the renderer.
- The large synchronization surface is `bundle_games`, not `bundle_keys`. Large bundles are common enough that the bundle-game sync must be incremental and cache-aware. Keep those rows in SQLite and expose targeted answers to the renderer instead of transferring the whole membership set through global app state.
- The API response is snake_case (`bundle_keys`, `created_at`, `games_count`, `cover_url`), while go-itchio structs use camelCase JSON tags. This is expected: go-itchio camelifies response maps before mapstructure decoding. Tests/fixtures should use real snake_case API payloads to catch regressions in the mapping.
- The sample payload does not include `owner_id`; that is fine. Like `DownloadKey`, butler should fill `BundleKey.OwnerID` locally by saving through `Profile.BundleKeys` with `hades:"foreign_key:owner_id"`.
- A profile can own multiple server-side bundle keys for the same `bundle_id` if they purchased or redeemed the same bundle more than once. The current go-itchio endpoint comment says `/profile/owned-bundles` is deduped across multiple purchases, so this fetch path may not receive every duplicate key. Do not add local uniqueness constraints on `(owner_id, bundle_id)`: if the API ever returns duplicate keys, persist them as real rows. UI-facing bundle lists and ownership summaries must still dedupe by `(owner_id, bundle_id)` or by `bundle_id`, depending on whether the response is profile-scoped.
- Bundle ownership status must not depend on whether the user has opened a specific bundle detail page. A profile-level sync should keep the owned bundle game list in SQLite, and targeted game ownership checks should read that local index. Do **not** put bundle-game ownership into `Fetch.Commons`: users can own thousands of bundle games, and commons is an always-on payload that is shuffled between butlerd and the renderer repeatedly.

## Resync policy

Butler's existing lazy fetch policy is TTL-based:

- `Fresh: false` never talks to itch.io. It reads local data and sets `Stale: true` when the corresponding `FetchTarget` is missing or older than its TTL.
- The caller then reissues the same request with `Fresh: true` to do the network refresh.
- `Fetch.ExpireAll` deletes all `fetch_info` rows and forces every target stale.

Bundle resync should follow that model:

1. `Fetch.ProfileOwnedBundles(fresh:false)` is cheap and should be called on app startup, login/profile switch, and app resume/focus. It returns cached bundle keys immediately and marks stale after `defaultTTL` (currently 2 minutes).
2. If stale, the renderer or reactor reissues `Fetch.ProfileOwnedBundles(fresh:true)`. This catches newly purchased bundles, removed/refunded bundle keys, and changed bundle metadata.
3. `Fetch.ProfileBundleOwnerships(fresh:false)` is the profile-level local sync status endpoint. It returns counts/status only, not the full game list. It is stale when `profile_owned_bundles` is stale or any owned bundle's `bundle_games` target is missing/stale.
4. `Fetch.ProfileBundleOwnerships(fresh:true)` refreshes `Fetch.ProfileOwnedBundles`, then ensures every distinct owned bundle's `bundle_games` target is fresh. This is the potentially expensive sync job, and it should run from deliberate/background library sync flows, not from high-frequency renderer interactions.
5. `Fetch.GameOwnership` is the small, targeted endpoint for direct game-page CTA state. It never talks to itch.io. It reads local `download_keys` first, then local `bundle_keys`/`bundle_games`, and returns a compact ownership/access summary for one game. It marks stale if the profile bundle ownership sync is stale or incomplete.
6. `Fetch.Commons` remains unchanged and local-only. It should continue to return materialized `DownloadKey` rows and installed cave state, but not bundle-game ownership.

So if a user buys a bundle in the browser and returns to the app, the maximum automatic detection delay for the owned bundle list is the `profile_owned_bundles` TTL unless the app explicitly forces refresh on focus. Direct game pages should not crawl remote bundles themselves; when they see stale ownership, they can show the cached/local answer and optionally trigger `Fetch.ProfileBundleOwnerships(fresh:true)` in the background or through an explicit refresh.

This does not need to block app startup. The current app already fetches commons from a reactor after `preboot`, `loginSucceeded`, focus changes, and install/uninstall events, and the initial owned-key fetch happens after `loginSucceeded` has already been dispatched. Bundle ownership should not add a startup-wide commons payload. Sync owned bundle headers and profile bundle ownership opportunistically in the background, and let direct game pages perform targeted local ownership checks.

Do not add special automatic invalidation based on `bundles.updated_at` for the first implementation. Bundle list pages, bundle detail pages, and direct game ownership checks should support the normal renderer refresh path (F5 / explicit fresh request), which resyncs the owned bundle list and relevant bundle games immediately. Otherwise, rely on `defaultTTL` for bundle keys and `longTTL` for bundle games.

## Non-materializing ownership: how the renderer will see ownership without claiming

A hard requirement: **viewing a game page must never trigger `ClaimBundleGame`**. Materialization is a write on itch.io's side (creates a real `DownloadKey` row); doing it speculatively on page view would create download keys for every bundle game the user ever browses past, defeating the whole reason ownership is deferred in the first place.

Verified flow in the desktop app (`/home/leafo/code/itch/itch`):

- `getGameStatus` (`src/common/helpers/get-game-status.ts:91`) currently reads ownership purely from the in-memory `commons` store. The decisive check is `commons.downloadKeyIdsByGameId[game.id]` — if anything is there, the CTA flips from "Buy" to "Install".
- `commons` is populated by `Fetch.Commons` (`src/main/reactors/commons.ts`) at boot and on a fixed list of reactive events (login, download end, uninstall end, etc.). It is **never** refetched on game-page navigation.
- Game page navigation only fires `Fetch.Game` (`src/renderer/pages/GamePage.tsx:15`). That does not touch ownership.
- The Install button click fires `Install.Queue` (`src/renderer/modal-widgets/PlanInstall/index.tsx:275`). This is the only path that needs to mutate ownership on the server.

So the design splits cleanly:

1. **View-path data**: The renderer keeps `Fetch.Commons` as the source for materialized `DownloadKey` ownership. On a direct game page, it also calls `Fetch.GameOwnership` for that one game/profile. The response says whether the game is owned directly, owned via a bundle, or not owned, plus the `bundleId` needed later for install. This is a small local per-game payload, not a global bundle ownership dump and not a remote membership crawl.
2. **Install-path materialization**: `Install.Queue` runs butler's existing `AccessForGameID` (purely local DB reads — never calls itch.io), and only at that point does the new materialization gate fire `ClaimBundleGame`.

`AccessForGameID` is the install/access credential resolver, and it makes zero network calls — it walks `download_keys`, then (new) `bundle_keys`/`bundle_games`, then falls back to a press/recent profile. The read-only ownership answer for the renderer comes from `Fetch.GameOwnership`. The Install.Queue gate is the *only* place a bundle key turns into a download key.

## Verified pre-conditions

- go-itchio has `Bundle`, `BundleGame`, `BundleKey` (`types.go`), `ListProfileOwnedBundles` (`endpoints_profile.go:77`), `GetBundleGames` (`endpoints_bundles.go:7`), `ClaimBundleGame` (`endpoints_bundles.go:31`).
- `BundleKey` carries an `OwnerID int64` field, matching `DownloadKey`. The API does not populate it; butler fills it locally by saving profile-owned bundle keys through a `Profile.BundleKeys` has-many association with `hades:"foreign_key:owner_id"`.
- `Bundle.BundleGames` is the inline children slice that must be nilled before persistence (the upstream comment already calls this out).
- `ClaimBundleGame` returns a `*ClaimBundleGameResponse` wrapper with a `.DownloadKey` field (not a bare `*DownloadKey`).
- `itchio.GameCredentials` has `DownloadKeyID`, `Password`, `Secret` and no bundle-aware field. Bundle materialization state lives on butler's local `GameAccess` struct instead, so the wire-level credentials struct stays unchanged.

## File-by-file changes

### 1. Register the new models

**`database/models/all_models.go`** (alongside existing Collection entries around lines 8-29):

```go
&itchio.Bundle{},
&itchio.BundleGame{},
&itchio.BundleKey{},
```

No new association struct needed (no `ProfileBundle` join): `BundleKey` carries `OwnerID` and is saved through a profile has-many association, exactly like `DownloadKey`/`Profile.OwnedKeys`.

### 2. Profile association

**`database/models/profile.go`** (next to `OwnedKeys`):

```go
BundleKeys []*itchio.BundleKey `json:"bundleKeys,omitempty" hades:"foreign_key:owner_id"`
```

This is the same pattern as:

```go
OwnedKeys []*itchio.DownloadKey `json:"ownedKeys,omitempty" hades:"foreign_key:owner_id"`
```

When `Fetch.ProfileOwnedBundles` saves a fake profile with `BundleKeys` populated, hades sets `bundle_keys.owner_id = profile.ID` before writing the child records.

### 3. Freshness targets

**`database/models/freshness.go`** (after the existing Collection equivalents at lines 42-48 and 74-80):

```go
func FetchTargetForProfileOwnedBundles(profileID int64) FetchTarget {
    return FetchTarget{ID: profileID, Type: "profile_owned_bundles", TTL: defaultTTL}
}

func FetchTargetForBundleGames(bundleID int64) FetchTarget {
    return FetchTarget{ID: bundleID, Type: "bundle_games", TTL: longTTL}
}

func FetchTargetForProfileBundleOwnerships(profileID int64) FetchTarget {
    return FetchTarget{ID: profileID, Type: "profile_bundle_ownerships", TTL: longTTL}
}
```

`longTTL` for bundle games and profile bundle ownerships matches `FetchTargetForCollectionGames` — bundles change infrequently and pulling thousands of pages on every refresh is wasteful.

### 4. New helper: `database/models/bundle_key_ext.go`

Helpers:

- `BundleKeysByGameID(conn *sqlite.Conn, gameID int64) []*itchio.BundleKey` — mirror `models.DownloadKeysByGameID`, but join `bundle_keys` to `bundle_games` on `bundle_id`. This can return multiple keys for the same profile/bundle if the user bought the same bundle more than once.
- `BundleIDOwningGameForProfile(conn *sqlite.Conn, gameID int64) (bundleID int64, profileID int64)` — return one owned bundle for the game by joining `bundle_games` to `bundle_keys` and reading `bundle_keys.owner_id`. The choice only needs to be deterministic enough to choose an API key/profile; `ClaimBundleGame` resolves the specific server-side key that makes sense.
- `ProfileOwnsGameViaBundle(conn *sqlite.Conn, profileID int64, gameID int64) bool` — same join, filtered by `bundle_keys.owner_id = profileID`.

All profile scoping is via `bundle_keys.owner_id`, just like owned-key queries use `download_keys.owner_id`.

### 5. Indexes

Bundle support adds joins that run on hot paths (`Fetch.GameOwnership`, `AccessForGameID`, bundle detail pages), so add explicit indexes instead of relying only on hades primary-key indexes.

Before adding the index migration, fix the existing migration loop in `database/models/migrations/migrations.go`: `Do` currently returns from inside the `for _, key := range todo` loop unconditionally after the first migration. The bundle index migration will be the second migration in this file, so without that fix it will silently never run on databases that also need the older migration. The loop should only return on error and continue through every pending migration before returning `nil`.

```sql
CREATE INDEX IF NOT EXISTS idx_bundle_keys_owner_id
ON bundle_keys(owner_id);

CREATE INDEX IF NOT EXISTS idx_bundle_keys_owner_bundle
ON bundle_keys(owner_id, bundle_id);

CREATE INDEX IF NOT EXISTS idx_bundle_games_game_bundle
ON bundle_games(game_id, bundle_id);
```

`bundle_games` should already have a composite primary key on `(bundle_id, game_id)` from `itchio.BundleGame`, which covers bundle detail page queries and `AssocReplace("BundleGames")`. If automigration does not create that index, add it explicitly too:

```sql
CREATE INDEX IF NOT EXISTS idx_bundle_games_bundle_game
ON bundle_games(bundle_id, game_id);
```

Access paths covered:

- `bundle_keys(owner_id)` — local owned-bundle list for a profile.
- `bundle_keys(owner_id, bundle_id)` — duplicate-purchase dedupe and distinct bundle IDs for ownership-index sync.
- `bundle_games(game_id, bundle_id)` — `AccessForGameID` and "does this profile own this game through any bundle?"
- `bundle_games(bundle_id, game_id)` — bundle detail pages and replacing a bundle's membership after a full sync.

### 6. butlerd request/response types

**`butlerd/types.go`** (mirror `FetchProfileCollections*` at lines 970-1032 and `FetchCollectionGames*` at lines 876-958):

```go
// @name Fetch.ProfileOwnedBundles
// @category Fetch
// @caller client
type FetchProfileOwnedBundlesParams struct {
    ProfileID int64  `json:"profileId"`
    Limit     int64  `json:"limit"`
    Cursor    Cursor `json:"cursor"`
    Fresh     bool   `json:"fresh"`
}
type FetchProfileOwnedBundlesResult struct {
    Items      []*itchio.BundleKey `json:"items"`
    NextCursor Cursor              `json:"nextCursor,omitempty"`
    Stale      bool                `json:"stale,omitempty"`
}

// @name Fetch.BundleGames
// @category Fetch
// @caller client
type FetchBundleGamesParams struct {
    ProfileID int64  `json:"profileId"`
    BundleID  int64  `json:"bundleId"`
    Limit     int64  `json:"limit"`
    Cursor    Cursor `json:"cursor"`
    Fresh     bool   `json:"fresh"`
}
type FetchBundleGamesResult struct {
    Items      []*itchio.BundleGame `json:"items"`
    NextCursor Cursor               `json:"nextCursor,omitempty"`
    Stale      bool                 `json:"stale,omitempty"`
}

// @name Fetch.GameOwnership
// @category Fetch
// @caller client
type FetchGameOwnershipParams struct {
    ProfileID int64 `json:"profileId"`
    GameID    int64 `json:"gameId"`
}
type FetchGameOwnershipResult struct {
    Owned         bool   `json:"owned"`
    DownloadKeyID int64  `json:"downloadKeyId,omitempty"`
    BundleID      int64  `json:"bundleId,omitempty"`
    Source        string `json:"source,omitempty"` // "download_key", "bundle", or empty
    Stale         bool   `json:"stale,omitempty"`
}

// @name Fetch.ProfileBundleOwnerships
// @category Fetch
// @caller client
type FetchProfileBundleOwnershipsParams struct {
    ProfileID int64 `json:"profileId"`
    Fresh     bool  `json:"fresh"`
}
type FetchProfileBundleOwnershipsResult struct {
    SyncedBundles int64 `json:"syncedBundles"`
    TotalBundles  int64 `json:"totalBundles"`
    Stale         bool  `json:"stale,omitempty"`
}
```

Plus the standard `Validate`, `GetProfileID`, `GetLimit`, `GetCursor`, `IsFresh`, and `SetStale` methods that the lazyfetch and pager packages require for paginated endpoints. `FetchGameOwnershipParams` only needs `Validate` and `GetProfileID`; it is always local-only. `FetchProfileBundleOwnershipsParams` needs `Validate`, `GetProfileID`, and `IsFresh`; its result needs `SetStale`.

These types will be picked up by the `generous` codegen for `messages.go` / `butlerd.json` on the next regen.

### 7. New handler: `endpoints/fetch/fetch_profileownedbundles.go`

Direct port of the owned-key save pattern plus the collection local paging pattern. Differences:

- Calls `client.ListProfileOwnedBundles(rc.Ctx)` and iterates `bundleKeysRes.BundleKeys`.
- Do not implement remote pagination here. `/profile/owned-bundles` returns the full owned-bundle-key list for the current API key and is capped server-side at 100. `Limit`/`Cursor` are only for local DB pagination back to the renderer.
- Stores keys directly on a fake profile (`fakeProfile.BundleKeys = pageKeys`) rather than wrapping in a `ProfileBundle` join. The `BundleKey` row is the relation, with `owner_id` filled by the profile association.
- **Critical**: before saving, walk the response and null `bk.Bundle.BundleGames = nil` on every key so the locally-paginated `bundle_games` table is not clobbered by partial inline data.
- Save mirrors owned keys:

```go
fakeProfile := &models.Profile{ID: profile.ID}
fakeProfile.BundleKeys = bundleKeysRes.BundleKeys
models.MustSave(conn, fakeProfile,
    hades.OmitRoot(),
    hades.AssocReplace("BundleKeys", hades.Assoc("Bundle")),
)
```

- UI-facing query must dedupe duplicate purchases of the same bundle. Return one `BundleKey` per `(owner_id, bundle_id)`, preferably the newest key (`max(bundle_keys.created_at)` / highest `bundle_keys.id`) so `acquiredAt` reflects the most recent acquisition. Do not delete older duplicate keys locally.
- Because the current API response is documented as deduped, `AssocReplace("BundleKeys")` mirrors the server-visible owned bundle list for that profile. That is acceptable for first implementation, since `/claim` resolves the concrete server-side key. The important local guarantee is that butler does not enforce a uniqueness constraint that would reject duplicate `BundleKey` rows if the API later returns them.
- Sort options: `acquiredAt`/default (`bundle_keys.created_at DESC`, matching the sample newest-first API order after dedupe), `title` (joining `bundles`), `updatedAt`, `gamesCount`. Search by `bundles.title`. Same shape as collections.

### 8. New handler: `endpoints/fetch/fetch_bundlegames.go`

Direct port of `endpoints/fetch/fetch_collectiongames.go`. The `LazyFetchBundleGames` helper:

- Loops `for page := int64(1); ; page++` calling `client.GetBundleGames(rc.Ctx, itchio.GetBundleGamesParams{BundleID, Page: page})`.
- Builds a `fakeBundle := &itchio.Bundle{ID: bundleID}` for the hades save.
- Saves each page incrementally with `hades.OmitRoot() + hades.Assoc("BundleGames", hades.Assoc("Game", hades.Assoc("Sale")))`.
- Final pass uses `hades.AssocReplace("BundleGames")`.
- Do **not** silently cap bundle-game sync below the bundle's `games_count`. Ownership inference depends on complete membership. If a hard safety cap is ever introduced, hitting it must leave `FetchTargetForBundleGames` stale/incomplete rather than marking a truncated membership set fresh.

### 9. New handler: `endpoints/fetch/fetch_gameownership.go`

This is the targeted local ownership check used by direct game-page CTA state. It never performs network I/O. Its job is to answer from the locally synchronized game list and clearly report whether that local answer may be stale.

Flow:

1. Always check materialized ownership first: `download_keys.owner_id = profileID AND game_id = params.GameID`. If found, return `Owned:true`, `Source:"download_key"`, `DownloadKeyID`, and `Stale:false`; direct download-key ownership does not depend on bundle sync freshness.
2. Check local bundle ownership next by joining `bundle_keys` to `bundle_games` for `(profileID, gameID)`. If found, return `Owned:true`, `Source:"bundle"`, and `BundleID`.
3. For bundle-sourced ownership, set `Stale:true` when `FetchTargetForProfileOwnedBundles(profileID)` or `FetchTargetForProfileBundleOwnerships(profileID)` is stale. Return the cached ownership answer anyway; stale cached positives are still useful UI state while a background sync runs.
4. If no local ownership exists and the profile bundle ownership target is stale, return `Owned:false`, `Stale:true`. This means "not locally known yet", not "definitively not owned."
5. If no local ownership exists and the profile bundle ownership target is fresh, return `Owned:false`, `Stale:false`.

### 10. New handler: `endpoints/fetch/fetch_profilebundleownerships.go`

This is the profile-level synchronization job that makes `Fetch.GameOwnership` useful without per-game API calls. It should return only progress/status counts, not the full game ownership list.

Flow:

1. On `Fresh:false`, return `Stale:true` if `FetchTargetForProfileBundleOwnerships(profileID)` or `FetchTargetForProfileOwnedBundles(profileID)` is stale, or if any distinct owned bundle has a stale/missing `FetchTargetForBundleGames(bundleID)`.
2. On `Fresh:true`, refresh `Fetch.ProfileOwnedBundles` first; the list is small and contains `bundle.games_count`.
3. Load distinct owned bundle IDs for `owner_id = params.ProfileID`; duplicate purchases of the same bundle should only be synced once.
4. For each owned bundle ID, ensure that bundle's `bundle_games` target is fresh by running the same page-walking helper used by `Fetch.BundleGames`.
5. Mark `FetchTargetForProfileBundleOwnerships(profileID)` fresh only after every owned bundle's games have been fetched successfully.
6. Return `SyncedBundles`, `TotalBundles`, and `Stale`; do not return `(gameId,bundleId)` rows.

Worst case, the first fresh sync for the sample account walks 15 bundle keys and persists about 4,458 `bundle_games` rows. That is acceptable as a deliberate profile sync/background job, and subsequent direct game ownership checks are local SQLite reads until TTL expiry.

### 11. Wire up the handlers

**`endpoints/fetch/fetch.go`** (in `Register`, alongside the existing collection registrations):

```go
messages.FetchProfileOwnedBundles.Register(router, FetchProfileOwnedBundles)
messages.FetchBundleGames.Register(router, FetchBundleGames)
messages.FetchGameOwnership.Register(router, FetchGameOwnership)
messages.FetchProfileBundleOwnerships.Register(router, FetchProfileBundleOwnerships)
```

### 12. Ownership inference: `cmd/operate/game_utils.go`

Add a profile-aware access resolver and a new fallback tier between the existing download-key block (lines 214-232) and the no-credentials fallback. Also add one new butler-local field to `GameAccess`:

```go
type GameAccess struct {
    APIKey      string                 `json:"api_key"`
    Credentials itchio.GameCredentials `json:"credentials"`
    ProfileID   int64                  `json:"profile_id,omitempty"` // owner profile used for later materialization persistence
    BundleID    int64                  `json:"bundle_id,omitempty"` // new: nonzero == owned via bundle, key not yet claimed
}
```

(Putting this on `GameAccess` rather than on `itchio.GameCredentials` keeps the wire-level credentials struct unchanged — bundle materialization is butler-local state, never transmitted to itch.io.)

Profile selection matters. `Fetch.GameOwnership` is profile-scoped, so an install launched from that UI should materialize using the same profile. Add a profile-aware helper, for example:

```go
func AccessForGameIDForProfile(conn *sqlite.Conn, gameID int64, profileID int64) *GameAccess
```

`AccessForGameID(conn, gameID)` can remain as the legacy "any suitable profile" wrapper for existing callers, but `Install.Queue` should accept/pass the active `ProfileID` when the renderer has one and use the profile-aware helper. Without this, the UI could show "owned via bundle" for profile A while install materializes through profile B if multiple profiles are cached locally.

The new bundle tier in the profile-aware resolver:

```go
// look for bundle ownership (deferred download key materialization)
{
    bundleID, profileID := models.BundleIDOwningGameForProfile(conn, gameID, preferredProfileID) // small new helper next to ProfileOwnsGameViaBundle
    if bundleID != 0 {
        profile := models.ProfileByID(conn, profileID)
        if profile != nil {
            return &GameAccess{
                APIKey:    profile.APIKey,
                ProfileID: profileID,
                BundleID:  bundleID,
            }
        }
    }
}
```

Subsequent installs/updates of this game will hit the materialization gate (next section), after which the game has a real `DownloadKey` and the download-key tier above will pick it up first.

### 13. Materialization gate: `endpoints/install/install_queue.go`

The materialization must happen before the first install-intent network call that needs owned credentials. In `Install.Queue`, do this with a small local helper (for example `ensureBundleAccessMaterialized`) called before upload/build listing or source URL construction:

- In the `params.Upload == nil` path, call it before `operate.GetFilteredUploads`, because that helper lists uploads.
- In the explicit-upload path, it can run after local validation/deduplication but before `client.ListGameUploads`, `client.GetBuild`, or download URL construction.

Inserting only before the later `client.ListGameUploads(...)` call is too late, because the no-explicit-upload path has already listed uploads through `GetFilteredUploads`.

```go
if params.Access.BundleID != 0 && params.Access.Credentials.DownloadKeyID == 0 {
    claimRes, err := client.ClaimBundleGame(rc.Ctx, itchio.ClaimBundleGameParams{
        BundleID: params.Access.BundleID,
        GameID:   params.Game.ID,
    })
    if err != nil {
        return nil, errors.WithStack(err)
    }
    dk := claimRes.DownloadKey

    // persist the new key the same way fetch_profileownedkeys.go does
    rc.WithConn(func(conn *sqlite.Conn) {
        fakeProfile := &models.Profile{ID: params.Access.ProfileID}
        fakeProfile.OwnedKeys = []*itchio.DownloadKey{dk}
        models.MustSave(conn, fakeProfile,
            hades.OmitRoot(),
            hades.Assoc("OwnedKeys", hades.Assoc("Game")),
        )
    })

    params.Access.Credentials.DownloadKeyID = dk.ID
    params.Access.BundleID = 0
}
```

`params.Access.ProfileID` comes from the bundle-key ownership lookup in `AccessForGameID`. The server never populates `OwnerID` on returned download keys or bundle keys; butler always fills it locally via the parent association's `hades:"foreign_key:owner_id"` tag. The `Profile.OwnedKeys` save above writes `download_keys.owner_id = params.Access.ProfileID`, matching `FetchProfileOwnedKeys` and the new `Profile.BundleKeys` flow.

After this gate the rest of the install flow is unchanged. **Update flow stays untouched** (`endpoints/update/update.go`) — "owned via bundle, key not yet claimed" is a legitimate state and we don't want update checks to trigger server round trips just to mint a key.

Other upload-listing endpoints need an explicit decision:

- `Fetch.GameUploads` stays non-materializing. It is a fetch/view endpoint and must not call `ClaimBundleGame`.
- `Install.Queue` must materialize before upload filtering/listing because it is the actual install mutation path.
- If the renderer uses `Game.FindUploads`, `Install.GetUploads`, or deprecated `Install.Plan` as part of the pre-queue install modal for bundle-owned paid games, those endpoints either need to call the same materialization helper or be bypassed for bundle-owned games until `Install.Queue`. Do not accidentally put claim logic inside `operate.GetFilteredUploads` unless every caller has been audited, because it is also used by non-queue endpoints.
- `InstallVersionSwitchQueue`, `Update.Check`, launch, and legacy location scan should not materialize unclaimed bundle ownership. In normal operation they deal with already-installed caves, which should already have a materialized download key if the install came from a bundle.

### 14. Renderer ownership plumbing

Do not add bundle ownership to `Fetch.Commons`. Commons is fetched and reshaped frequently by the frontend, and bundle ownership can easily be thousands of rows. Keep commons focused on small global state: materialized download keys, caves, and install locations.

For direct game pages, the renderer should call `Fetch.GameOwnership` after or alongside `Fetch.Game` for the current game/profile. The response is intentionally small:

```go
type FetchGameOwnershipResult struct {
    Owned         bool   `json:"owned"`
    DownloadKeyID int64  `json:"downloadKeyId,omitempty"`
    BundleID      int64  `json:"bundleId,omitempty"`
    Source        string `json:"source,omitempty"` // "download_key", "bundle", or empty
    Stale         bool   `json:"stale,omitempty"`
}
```

Renderer behavior:

- Existing library/listing surfaces can continue using `Fetch.Commons` and therefore only show materialized download-key ownership.
- Direct game pages layer `Fetch.GameOwnership` on top of `getGameStatus` so a bundle-owned game can show installable state without adding every bundle game to commons.
- Bundle detail pages know they are in an owned bundle context and can use `Fetch.BundleGames` for paginated contents; they do not need commons to contain the whole bundle ownership set.
- When `Fetch.GameOwnership` returns `Stale:true`, the renderer may trigger `Fetch.ProfileBundleOwnerships(fresh:true)` in a debounced/background path, then re-read `Fetch.GameOwnership`. It should not call profile bundle sync from high-frequency surfaces like search keystrokes, hovers, or every list row.

This means bundle ownership can be complete enough for the current view while avoiding a global `bundleGameIdsByGameId` map in renderer memory.

## Critical files

- `database/models/all_models.go` — register types
- `database/models/freshness.go` — TTL targets
- `database/models/migrations/migrations.go` — fix migration loop, then add explicit bundle lookup indexes
- `database/models/bundle_key_ext.go` — **new**, ownership helpers
- `butlerd/types.go` — RPC params/results for bundle endpoints + `FetchGameOwnership`
- `endpoints/fetch/fetch_profileownedbundles.go` — **new**
- `endpoints/fetch/fetch_bundlegames.go` — **new**
- `endpoints/fetch/fetch_gameownership.go` — **new**
- `endpoints/fetch/fetch_profilebundleownerships.go` — **new**, profile-level bundle game sync status/job
- `endpoints/fetch/fetch.go` — handler registration
- `cmd/operate/game_utils.go` — `GameAccess.BundleID`, profile-aware access resolver, third tier in `AccessForGameID`
- `butlerd/types.go` / `endpoints/install/install_queue.go` — pass active profile into install when available; materialization gate before install-intent upload/build listing

## Reused existing utilities

- `lazyfetch.Do` and `lazyfetch.Targets` (`endpoints/fetch/lazyfetch/`) for cache-aware fetch
- `pager.New` and `pager.Ordering` (`endpoints/fetch/pager/`) for pagination
- `hades.Assoc`, `hades.AssocReplace`, `hades.OmitRoot` for persistence
- `models.MustSave`, `models.MustPreload`, `models.MustExec`, `models.Must` (existing patterns from `fetch_profilecollections.go`, `fetch_collectiongames.go`, `fetch_profileownedkeys.go`)
- `rc.ProfileClient` for authenticated client construction

## Verification

1. **Build and unit tests**: `go build ./...` then `go test ./...` from `/home/leafo/code/itch/butler`.
2. **Codegen sanity**: regenerate `butlerd/messages.go` / `butlerd/generous/spec/butlerd.json` (whatever the existing `generous` invocation is) and confirm the four new RPCs appear: `Fetch.ProfileOwnedBundles`, `Fetch.BundleGames`, `Fetch.GameOwnership`, and `Fetch.ProfileBundleOwnerships`.
3. **Live integration**: with a profile that owns a multi-game bundle (e.g. an old Bundle for Racial Justice account on staging), exercise:
   - `Fetch.ProfileOwnedBundles` → returns bundle keys with `Bundle` populated; second call within `defaultTTL` is served from cache (`Stale: false` first, then cached).
   - Buy or grant a new bundle outside the app, wait until `defaultTTL` expires or force `Fetch.ProfileOwnedBundles(fresh:true)`, and confirm the new `bundle_key` appears locally with `owner_id = profile.ID`.
   - `Fetch.BundleGames` with paging → walks all pages, persists incrementally; `bundle_games` table populated; second call uses `longTTL` cache.
   - `Fetch.ProfileBundleOwnerships(fresh:false)` → reports stale when the profile's bundle game list is incomplete/stale, without returning bundle-game rows.
   - `Fetch.ProfileBundleOwnerships(fresh:true)` → refreshes owned bundles, walks every distinct owned bundle, persists about `sum(bundle.games_count)` local `bundle_games` rows, returns only sync counts/status, and marks the profile ownership target fresh only after success.
   - `Fetch.GameOwnership` for a bundle-owned game after sync → performs no HTTP calls and returns `Owned:true`, `Source:"bundle"`, and `BundleID`.
   - `Fetch.GameOwnership` for a game not present in any owned bundle after sync → performs no HTTP calls, returns `Owned:false`, `Stale:false`, and does not add anything to `download_keys`.
   - `Fetch.GameOwnership` before/after TTL expiry → returns cached ownership if present and sets `Stale:true` when the profile bundle ownership sync is stale.
   - `Fetch.Commons` remains unchanged: no bundle ownership field and no bulk bundle-game payload.
   - **Non-materializing view path**: with HTTP traffic logging on, fire `Fetch.Game` and `Fetch.GameOwnership` for ten different bundle-owned games and confirm zero `POST /bundles/:id/claim-game` calls. Then confirm the local `download_keys` table did not gain any rows.
   - Profile-aware access for a game-via-bundle returns access with `BundleID != 0`, the requested owning `ProfileID`, and `Credentials.DownloadKeyID == 0` — and confirm via traffic logs that this read fired no HTTP calls.
   - `Install.Queue` for that game triggers exactly one `POST /bundles/:id/claim-game`, materializes a `DownloadKey` (visible in `download_keys` and surfaced in the next `Fetch.Commons` poll under `DownloadKeys`), and proceeds with normal install.
   - With two cached profiles, install from profile A's bundle ownership materializes and saves the resulting `DownloadKey` with `owner_id = profileA.ID`, not whichever profile `AccessForGameID` would have picked globally.
   - Re-running install on the same game does not re-claim (idempotent on server, and locally the second pass sees `DownloadKeyID != 0`).
   - `Update.Check` for a still-unclaimed bundle game does not trigger a `ClaimBundleGame` call.
4. **Schema migration**: confirm hades emits the three new tables (`bundles`, `bundle_games`, `bundle_keys`) on butlerd startup against an existing user DB, and confirm the explicit lookup indexes exist (`idx_bundle_keys_owner_id`, `idx_bundle_keys_owner_bundle`, `idx_bundle_games_game_bundle`, and `idx_bundle_games_bundle_game` if needed). Add a regression check for `migrations.Do` with two pending migrations so the loop bug cannot reappear.

## Out of scope (separate issue)

- Renderer `getGameStatus`, `commons` reducer, library UI for bundle attribution
- Bundle-aware UI (badges, "Owned via bundles" library tab, bundle detail pages)
